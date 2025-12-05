package server

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/eager7/dogd/btcec"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
	v1 "github.com/vultisig/commondata/go/vultisig/vault/v1"
	"github.com/vultisig/mobile-tss-lib/tss"
	vv "github.com/vultisig/verifier/common/vultisig_validator"
	"github.com/vultisig/verifier/plugin"
	"github.com/vultisig/verifier/plugin/metrics"
	"github.com/vultisig/verifier/plugin/policy"
	"github.com/vultisig/verifier/plugin/redis"
	"github.com/vultisig/verifier/plugin/tasks"
	vtypes "github.com/vultisig/verifier/types"
	"github.com/vultisig/verifier/vault"
	vcommon "github.com/vultisig/vultisig-go/common"
	vgtypes "github.com/vultisig/vultisig-go/types"
	"golang.org/x/sync/errgroup"
	"google.golang.org/protobuf/encoding/protojson"
)

type Server struct {
	cfg            Config
	redis          *redis.Redis
	vaultStorage   vault.Storage
	client         *asynq.Client
	inspector      *asynq.Inspector
	policy         policy.Service
	spec           plugin.Spec
	logger         *logrus.Logger
	middlewares    []echo.MiddlewareFunc
	authMiddleware echo.MiddlewareFunc
	metrics        metrics.PluginServerMetrics
}

// NewServer returns a new server.
func NewServer(
	cfg Config,
	policy policy.Service,
	redis *redis.Redis,
	vaultStorage vault.Storage,
	client *asynq.Client,
	inspector *asynq.Inspector,
	spec plugin.Spec,
	middlewares []echo.MiddlewareFunc,
	metrics metrics.PluginServerMetrics, // Optional: pass nil to disable metrics
) *Server {
	return &Server{
		cfg:          cfg,
		redis:        redis,
		client:       client,
		inspector:    inspector,
		vaultStorage: vaultStorage,
		spec:         spec,
		logger:       logrus.WithField("pkg", "server").Logger,
		policy:       policy,
		middlewares:  middlewares,
		metrics:      metrics,
	}
}

func (s *Server) SetAuthMiddleware(auth echo.MiddlewareFunc) {
	s.authMiddleware = auth
}

func (s *Server) Start(ctx context.Context) error {
	e := echo.New()

	// Prepare middlewares slice with optional metrics middleware
	middlewares := s.middlewares
	if s.metrics != nil {
		// Prepend metrics middleware to the beginning of the slice
		middlewares = append([]echo.MiddlewareFunc{s.metrics.HTTPMiddleware()}, middlewares...)
	}

	// Add all middlewares
	e.Use(middlewares...)

	e.Validator = &vv.VultisigValidator{Validator: validator.New()}

	e.GET("/healthz", s.handleHealthz)

	vlt := e.Group("/vault")
	vlt.POST("/reshare", s.handleReshareVault)
	vlt.GET("/get/:pluginId/:publicKeyECDSA", s.handleGetVault)
	vlt.GET("/exist/:pluginId/:publicKeyECDSA", s.handleExistVault)
	vlt.GET("/sign/response/:taskId", s.handleGetKeysignResult)
	vlt.DELETE("/:pluginId/:publicKeyECDSA", s.handleDeleteVault)

	plg := e.Group("/plugin", s.VerifierAuthMiddleware)
	plg.POST("/policy", s.handleCreatePluginPolicy)
	plg.PUT("/policy", s.handleUpdatePluginPolicyById)
	plg.GET("/recipe-specification", s.handleGetRecipeSpecification)
	plg.POST("/recipe-specification/suggest", s.handleGetRecipeSpecificationSuggest)
	plg.DELETE("/policy/:policyId", s.handleDeletePluginPolicyById)

	eg := &errgroup.Group{}
	eg.Go(func() error {
		err := e.Start(fmt.Sprintf(":%d", s.cfg.Port))
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("failed to start server: %w", err)
	})
	eg.Go(func() error {
		<-ctx.Done()
		s.logger.Info("shutting down server...")

		c, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()

		err := e.Shutdown(c)
		if err != nil {
			return fmt.Errorf("failed to shutdown server: %w", err)
		}
		return nil
	})

	return eg.Wait()
}

func (s *Server) handleHealthz(c echo.Context) error {
	return c.String(http.StatusOK, "plugin server is running")
}

// ReshareVault is a handler to reshare a vault
func (s *Server) handleReshareVault(c echo.Context) error {
	var req vtypes.ReshareRequest
	if err := c.Bind(&req); err != nil {
		return fmt.Errorf("fail to parse request, err: %w", err)
	}
	if err := req.IsValid(); err != nil {
		return fmt.Errorf("invalid request, err: %w", err)
	}
	buf, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("fail to marshal to json, err: %w", err)
	}
	result, err := s.redis.Get(c.Request().Context(), req.SessionID)
	if err == nil && result != "" {
		return c.NoContent(http.StatusOK)
	}

	if err := s.redis.Set(c.Request().Context(), req.SessionID, req.SessionID, 5*time.Minute); err != nil {
		s.logger.Errorf("fail to set session, err: %v", err)
	}
	_, err = s.client.Enqueue(asynq.NewTask(tasks.TypeReshareDKLS, buf),
		asynq.MaxRetry(-1),
		asynq.Timeout(7*time.Minute),
		asynq.Retention(10*time.Minute),
		asynq.Queue(tasks.QUEUE_NAME))
	if err != nil {
		return fmt.Errorf("fail to enqueue task, err: %w", err)
	}
	return c.NoContent(http.StatusOK)
}

func (s *Server) handleGetVault(c echo.Context) error {
	publicKeyECDSA := c.Param("publicKeyECDSA")
	if publicKeyECDSA == "" {
		return c.JSON(http.StatusBadRequest, NewErrorResponse("public key is required"))
	}
	if !s.isValidHash(publicKeyECDSA) {
		return c.NoContent(http.StatusBadRequest)
	}
	pluginId := c.Param("pluginId")
	if pluginId == "" {
		return c.JSON(http.StatusBadRequest, NewErrorResponse("pluginId is required"))
	}

	filePathName := vcommon.GetVaultBackupFilename(publicKeyECDSA, pluginId)
	content, err := s.vaultStorage.GetVault(filePathName)
	if err != nil {
		wrappedErr := fmt.Errorf("fail to read file in GetVault, err: %w", err)
		s.logger.Error(wrappedErr)
		return wrappedErr
	}

	v, err := vcommon.DecryptVaultFromBackup(s.cfg.EncryptionSecret, content)
	if err != nil {
		s.logger.WithError(err).Error("fail to decrypt vault")
		return c.JSON(http.StatusInternalServerError, NewErrorResponse("fail to get vault"))
	}

	return c.JSON(http.StatusOK, vgtypes.VaultGetResponse{
		Name:           v.Name,
		PublicKeyEcdsa: v.PublicKeyEcdsa,
		PublicKeyEddsa: v.PublicKeyEddsa,
		HexChainCode:   v.HexChainCode,
		LocalPartyId:   v.LocalPartyId,
	})
}

// GetKeysignResult is a handler to get the keysign response
func (s *Server) handleGetKeysignResult(c echo.Context) error {
	taskID := c.Param("taskId")
	if taskID == "" {
		return fmt.Errorf("task id is required")
	}
	result, err := tasks.GetTaskResult(s.inspector, taskID)
	if err != nil {
		if err.Error() == "task is still in progress" {
			return c.JSON(http.StatusOK, "Task is still in progress")
		}
		return err
	}

	return c.JSON(http.StatusOK, result)
}

func (s *Server) handleDeleteVault(c echo.Context) error {
	publicKeyECDSA := c.Param("publicKeyECDSA")
	if publicKeyECDSA == "" {
		return c.JSON(http.StatusBadRequest, NewErrorResponse("public key is required"))
	}
	if !s.isValidHash(publicKeyECDSA) {
		return c.NoContent(http.StatusBadRequest)
	}
	pluginId := c.Param("pluginId")
	if pluginId == "" {
		return c.JSON(http.StatusBadRequest, NewErrorResponse("pluginId is required"))
	}

	fileName := vcommon.GetVaultBackupFilename(publicKeyECDSA, pluginId)
	if err := s.vaultStorage.DeleteFile(fileName); err != nil {
		// Check if it's a "file not found" error
		if os.IsNotExist(err) {
			// File doesn't exist - deletion "succeeded" (idempotent)
			return c.NoContent(http.StatusOK)
		}
		// Real error (S3 failure, permissions, etc.)
		return c.JSON(http.StatusInternalServerError, NewErrorResponse(err.Error()))
	}
	return c.NoContent(http.StatusOK)
}

func (s *Server) isValidHash(hash string) bool {
	if len(hash) != 66 {
		return false
	}
	_, err := hex.DecodeString(hash)
	return err == nil
}

func (s *Server) handleExistVault(c echo.Context) error {
	publicKeyECDSA := c.Param("publicKeyECDSA")
	if publicKeyECDSA == "" {
		return c.JSON(http.StatusBadRequest, NewErrorResponse("public key is required"))
	}
	if !s.isValidHash(publicKeyECDSA) {
		return c.NoContent(http.StatusBadRequest)
	}
	pluginId := c.Param("pluginId")
	if pluginId == "" {
		return c.JSON(http.StatusBadRequest, NewErrorResponse("spec id is required"))
	}

	filePathName := vcommon.GetVaultBackupFilename(publicKeyECDSA, pluginId)
	exist, err := s.vaultStorage.Exist(filePathName)
	if err != nil || !exist {
		return c.NoContent(http.StatusBadRequest)
	}
	return c.NoContent(http.StatusOK)
}

func (s *Server) handleCreatePluginPolicy(c echo.Context) error {
	var pol vtypes.PluginPolicy
	if err := c.Bind(&pol); err != nil {
		return fmt.Errorf("fail to parse request, err: %w", err)
	}

	if err := s.spec.ValidatePluginPolicy(pol); err != nil {
		s.logger.WithError(err).Error("failed to validate spec policy")
		return c.JSON(http.StatusBadRequest, NewErrorResponse("failed to validate policy"))
	}

	if pol.ID.String() == "" {
		pol.ID = uuid.New()
	}

	if !s.verifyPolicySignature(pol) {
		s.logger.Error("invalid policy signature")
		return c.JSON(http.StatusForbidden, NewErrorResponse("Invalid policy signature"))
	}

	newPolicy, err := s.policy.CreatePolicy(c.Request().Context(), pol)
	if err != nil {
		s.logger.WithError(err).Error("Failed to create spec policy")
		return c.JSON(http.StatusInternalServerError, NewErrorResponse("failed to create policy"))
	}

	return c.JSON(http.StatusOK, newPolicy)
}

func (s *Server) handleUpdatePluginPolicyById(c echo.Context) error {
	var pol vtypes.PluginPolicy
	if err := c.Bind(&pol); err != nil {
		return fmt.Errorf("fail to parse request, err: %w", err)
	}

	if err := s.spec.ValidatePluginPolicy(pol); err != nil {
		s.logger.WithError(err).
			WithField("plugin_id", pol.PluginID).
			WithField("policy_id", pol.ID).
			Error("Failed to validate spec policy")
		return c.JSON(http.StatusBadRequest, NewErrorResponse("failed to validate policy"))
	}

	if !s.verifyPolicySignature(pol) {
		s.logger.Error("invalid policy signature")
		return c.JSON(http.StatusForbidden, NewErrorResponse("Invalid policy signature"))
	}

	updatedPolicy, err := s.policy.UpdatePolicy(c.Request().Context(), pol)
	if err != nil {
		s.logger.WithError(err).Error("Failed to update spec policy")
		return c.JSON(http.StatusInternalServerError, NewErrorResponse("failed to update policy"))
	}

	return c.JSON(http.StatusOK, updatedPolicy)
}

func (s *Server) handleDeletePluginPolicyById(c echo.Context) error {
	var reqBody struct {
		Signature string `json:"signature"`
	}

	if err := c.Bind(&reqBody); err != nil {
		return c.JSON(http.StatusBadRequest, NewErrorResponse("fail to parse request"))
	}

	policyID := c.Param("policyId")
	if policyID == "" {
		return c.JSON(http.StatusBadRequest, NewErrorResponse("invalid policy ID"))
	}
	uPolicyID, err := uuid.Parse(policyID)
	if err != nil {
		s.logger.WithError(err).
			WithField("policy_id", policyID).
			Error("Failed to parse policy ID")
		return c.JSON(http.StatusBadRequest, NewErrorResponse("invalid policy ID"))
	}
	policy, err := s.policy.GetPluginPolicy(c.Request().Context(), uPolicyID)
	if err != nil {
		s.logger.WithError(err).
			WithField("policy_id", policyID).
			Error("Failed to get spec policy")
		return c.JSON(http.StatusInternalServerError, NewErrorResponse("failed to get policy"))
	}

	// This is because we have different signature stored in the database.
	policy.Signature = reqBody.Signature

	if !s.verifyPolicySignature(*policy) {
		return c.JSON(http.StatusForbidden, NewErrorResponse("Invalid policy signature"))
	}

	if err := s.policy.DeletePolicy(c.Request().Context(), uPolicyID, reqBody.Signature); err != nil {
		s.logger.WithError(err).
			WithField("policy_id", policyID).
			Error("Failed to delete spec policy")
		return c.JSON(http.StatusInternalServerError, NewErrorResponse("failed to delete policy"))
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"policy_id": policyID,
	})
}

func (s *Server) handleGetRecipeSpecification(c echo.Context) error {
	recipeSpec, err := s.spec.GetRecipeSpecification()
	if err != nil {
		s.logger.WithError(err).Error("Failed to get recipe spec")
		return c.JSON(http.StatusInternalServerError, NewErrorResponse("failed to get recipe spec"))
	}
	return c.JSON(http.StatusOK, recipeSpec)
}

func (s *Server) handleGetRecipeSpecificationSuggest(c echo.Context) error {
	type reqBody struct {
		Configuration map[string]any `json:"configuration"`
	}
	var req reqBody
	err := c.Bind(&req)
	if err != nil {
		s.logger.WithError(err).Error("Failed to parse request")
		return c.JSON(http.StatusInternalServerError, NewErrorResponse("failed to parse request"))
	}

	recipeSpec, err := s.spec.Suggest(req.Configuration)
	if err != nil {
		s.logger.WithError(err).Error("Failed to suggest recipe spec")
		return c.JSON(http.StatusInternalServerError, NewErrorResponse("failed to suggest recipe spec"))
	}

	b, err := protojson.Marshal(recipeSpec)
	if err != nil {
		s.logger.WithError(err).Error("Failed to proto-marshal recipe spec")
		return c.JSON(http.StatusInternalServerError, NewErrorResponse("failed to proto-marshal recipe spec"))
	}

	var r map[string]interface{}
	err = json.Unmarshal(b, &r)
	if err != nil {
		s.logger.WithError(err).Error("Failed to unmarshal recipe spec")
		return c.JSON(http.StatusInternalServerError, NewErrorResponse("failed to unmarshal recipe spec"))
	}

	return c.JSON(http.StatusOK, r)
}

func (s *Server) verifyPolicySignature(policy vtypes.PluginPolicy) bool {
	msgBytes, err := policyToMessageHex(policy)
	if err != nil {
		s.logger.WithError(err).Error("Failed to convert policy to message hex")
		return false
	}

	signatureBytes, err := hex.DecodeString(strings.TrimPrefix(policy.Signature, "0x"))
	if err != nil {
		s.logger.WithError(err).Error("Failed to decode signature bytes")
		return false
	}
	vault, err := s.getVault(policy.PublicKey, policy.PluginID.String())
	if err != nil {
		s.logger.WithError(err).Error("fail to get vault")
		return false
	}
	derivedPublicKey, err := tss.GetDerivedPubKey(vault.PublicKeyEcdsa, vault.HexChainCode, vcommon.Ethereum.GetDerivePath(), false)
	if err != nil {
		s.logger.WithError(err).Error("failed to get derived public key")
		return false
	}

	isVerified, err := verifyPolicySignature(derivedPublicKey, msgBytes, signatureBytes)
	if err != nil {
		s.logger.WithError(err).Error("Failed to verify signature")
		return false
	}
	return isVerified
}

func (s *Server) getVault(publicKeyECDSA, pluginId string) (*v1.Vault, error) {
	if len(s.cfg.EncryptionSecret) == 0 {
		return nil, fmt.Errorf("no encryption secret")
	}
	fileName := vcommon.GetVaultBackupFilename(publicKeyECDSA, pluginId)
	vaultContent, err := s.vaultStorage.GetVault(fileName)
	if err != nil {
		s.logger.WithError(err).Error("fail to get vault")
		return nil, fmt.Errorf("failed to get vault, err: %w", err)
	}

	v, err := vcommon.DecryptVaultFromBackup(s.cfg.EncryptionSecret, vaultContent)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt vault,err: %w", err)
	}
	return v, nil
}

func verifyPolicySignature(publicKeyHex string, messageHex []byte, signature []byte) (bool, error) {
	msgHash := crypto.Keccak256([]byte(fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(messageHex), messageHex)))
	publicKeyBytes, err := hex.DecodeString(publicKeyHex)
	if err != nil {
		return false, err
	}

	pk, err := btcec.ParsePubKey(publicKeyBytes, btcec.S256())
	if err != nil {
		return false, err
	}

	ecdsaPubKey := ecdsa.PublicKey{
		Curve: btcec.S256(),
		X:     pk.X,
		Y:     pk.Y,
	}
	R := new(big.Int).SetBytes(signature[:32])
	S := new(big.Int).SetBytes(signature[32:64])

	return ecdsa.Verify(&ecdsaPubKey, msgHash, R, S), nil
}

// policyToMessageHex converts a spec policy to a message hex string for signature verification.
// It joins policy fields with a delimiter and validates that no field contains the delimiter.
func policyToMessageHex(policy vtypes.PluginPolicy) ([]byte, error) {
	delimiter := "*#*"
	fields := []string{
		policy.Recipe,
		policy.PublicKey,
		fmt.Sprintf("%d", policy.PolicyVersion),
		policy.PluginVersion}
	for _, item := range fields {
		if strings.Contains(item, delimiter) {
			return nil, fmt.Errorf("invalid policy signature")
		}
	}
	result := strings.Join(fields, delimiter)
	return []byte(result), nil
}
