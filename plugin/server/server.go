package server

import (
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/DataDog/datadog-go/statsd"
	"github.com/eager7/dogd/btcec"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	"github.com/sirupsen/logrus"
	v1 "github.com/vultisig/commondata/go/vultisig/vault/v1"
	"github.com/vultisig/mobile-tss-lib/tss"
	vcommon "github.com/vultisig/verifier/common"
	vv "github.com/vultisig/verifier/common/vultisig_validator"
	"github.com/vultisig/verifier/plugin"
	"github.com/vultisig/verifier/plugin/policy"
	"github.com/vultisig/verifier/plugin/redis"
	"github.com/vultisig/verifier/plugin/tasks"
	vtypes "github.com/vultisig/verifier/types"
	"github.com/vultisig/verifier/vault"
)

type Server struct {
	cfg          Config
	redis        *redis.Redis
	vaultStorage vault.Storage
	client       *asynq.Client
	inspector    *asynq.Inspector
	sdClient     *statsd.Client
	policy       policy.Service
	spec         plugin.Spec
	logger       *logrus.Logger
	mode         string
}

// NewServer returns a new server.
func NewServer(
	cfg Config,
	policy policy.Service,
	redis *redis.Redis,
	vaultStorage vault.Storage,
	client *asynq.Client,
	inspector *asynq.Inspector,
	sdClient *statsd.Client,
	spec plugin.Spec,
) *Server {
	return &Server{
		cfg:          cfg,
		redis:        redis,
		client:       client,
		inspector:    inspector,
		sdClient:     sdClient,
		vaultStorage: vaultStorage,
		spec:         spec,
		logger:       logrus.WithField("pkg", "server").Logger,
		policy:       policy,
	}
}

func (s *Server) StartServer() error {
	e := echo.New()
	e.Logger.SetLevel(log.DEBUG)
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.BodyLimit("2M"))
	e.Use(s.statsdMiddleware)
	e.Use(middleware.CORS())
	limiterStore := middleware.NewRateLimiterMemoryStoreWithConfig(
		middleware.RateLimiterMemoryStoreConfig{Rate: 5, Burst: 30, ExpiresIn: 5 * time.Minute},
	)
	e.Use(middleware.RateLimiter(limiterStore))

	e.Validator = &vv.VultisigValidator{Validator: validator.New()}

	e.GET("/healthz", s.Healthz)

	vlt := e.Group("/vault")
	vlt.POST("/reshare", s.ReshareVault)
	vlt.GET("/get/:pluginId/:publicKeyECDSA", s.GetVault)
	vlt.GET("/exist/:pluginId/:publicKeyECDSA", s.ExistVault)
	vlt.GET("/sign/response/:taskId", s.GetKeysignResult)
	vlt.DELETE("/:pluginId/:publicKeyECDSA", s.DeleteVault)

	plg := e.Group("/plugin")
	plg.POST("/policy", s.CreatePluginPolicy)
	plg.PUT("/policy", s.UpdatePluginPolicyById)
	plg.GET("/recipe-specification", s.GetRecipeSpecification)
	plg.DELETE("/policy/:policyId", s.DeletePluginPolicyById)

	return e.Start(fmt.Sprintf(":%d", s.cfg.Port))
}

func (s *Server) statsdMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		start := time.Now()
		err := next(c)
		duration := time.Since(start).Milliseconds()

		// Send metrics to statsd
		_ = s.sdClient.Incr("http.requests", []string{"path:" + c.Path()}, 1)
		_ = s.sdClient.Timing("http.response_time", time.Duration(duration)*time.Millisecond, []string{"path:" + c.Path()}, 1)
		_ = s.sdClient.Incr("http.status."+fmt.Sprint(c.Response().Status), []string{"path:" + c.Path(), "method:" + c.Request().Method}, 1)

		return err
	}
}

func (s *Server) Healthz(c echo.Context) error {
	return c.String(http.StatusOK, "Payroll & DCA Plugin server is running")
}

// ReshareVault is a handler to reshare a vault
func (s *Server) ReshareVault(c echo.Context) error {
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

func (s *Server) GetVault(c echo.Context) error {
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

	return c.JSON(http.StatusOK, vtypes.VaultGetResponse{
		Name:           v.Name,
		PublicKeyEcdsa: v.PublicKeyEcdsa,
		PublicKeyEddsa: v.PublicKeyEddsa,
		HexChainCode:   v.HexChainCode,
		LocalPartyId:   v.LocalPartyId,
	})
}

// SignMessages is a handler to process Keysing request
func (s *Server) SignMessages(c echo.Context) error {
	s.logger.Debug("VERIFIER SERVER: SIGN MESSAGES")
	var req vtypes.KeysignRequest
	if err := c.Bind(&req); err != nil {
		return fmt.Errorf("fail to parse request, err: %w", err)
	}
	if err := req.IsValid(); err != nil {
		return fmt.Errorf("invalid request, err: %w", err)
	}
	if !s.isValidHash(req.PublicKey) {
		return c.NoContent(http.StatusBadRequest)
	}
	result, err := s.redis.Get(c.Request().Context(), req.SessionID)
	if err == nil && result != "" {
		return c.NoContent(http.StatusOK)
	}

	if err := s.redis.Set(c.Request().Context(), req.SessionID, req.SessionID, 30*time.Minute); err != nil {
		s.logger.Errorf("fail to set session, err: %v", err)
	}

	filePathName := vcommon.GetVaultBackupFilename(req.PublicKey, req.PluginID)
	content, err := s.vaultStorage.GetVault(filePathName)
	if err != nil {
		wrappedErr := fmt.Errorf("fail to read file in SignMessages, err: %w", err)
		s.logger.Infof("fail to read file in SignMessages, err: %v", err)
		s.logger.Error(wrappedErr)
		return wrappedErr
	}

	_, err = vcommon.DecryptVaultFromBackup(s.cfg.EncryptionSecret, content)
	if err != nil {
		return fmt.Errorf("fail to decrypt vault from the backup, err: %w", err)
	}
	buf, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("fail to marshal to json, err: %w", err)
	}

	ti, err := s.client.EnqueueContext(c.Request().Context(),
		asynq.NewTask(tasks.TypeKeySignDKLS, buf),
		asynq.MaxRetry(-1),
		asynq.Timeout(2*time.Minute),
		asynq.Retention(5*time.Minute),
		asynq.Queue(tasks.QUEUE_NAME))

	if err != nil {
		return fmt.Errorf("fail to enqueue task, err: %w", err)
	}

	return c.JSON(http.StatusOK, ti.ID)

}

// GetKeysignResult is a handler to get the keysign response
func (s *Server) GetKeysignResult(c echo.Context) error {
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
func (s *Server) DeleteVault(c echo.Context) error {
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

func (s *Server) ExistVault(c echo.Context) error {
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

func (s *Server) GetPluginPolicyById(c echo.Context) error {
	policyID := c.Param("policyId")
	if policyID == "" {
		return c.JSON(http.StatusBadRequest, NewErrorResponse("invalid policy ID"))
	}
	uPolicyID, err := uuid.Parse(policyID)
	if err != nil {
		s.logger.WithError(err).
			WithField("policy_id", policyID).
			Error("failed to parse policy ID")
		return c.JSON(http.StatusBadRequest, NewErrorResponse("invalid policy ID"))
	}
	policy, err := s.policy.GetPluginPolicy(c.Request().Context(), uPolicyID)
	if err != nil {
		s.logger.WithError(err).
			WithField("policy_id", policyID).
			Error("fail to get policy from database")
		return c.JSON(http.StatusInternalServerError, NewErrorResponse("failed to get policy"))
	}

	return c.JSON(http.StatusOK, policy)
}

func (s *Server) GetAllPluginPolicies(c echo.Context) error {
	publicKey := c.Request().Header.Get("public_key")
	if publicKey == "" {
		return c.JSON(http.StatusBadRequest, NewErrorResponse("missing required header: public_key"))
	}

	pluginID := c.Request().Header.Get("plugin_id")
	if pluginID == "" {
		return c.JSON(http.StatusBadRequest, NewErrorResponse("missing required header: plugin_id"))
	}

	policies, err := s.policy.GetPluginPolicies(c.Request().Context(), vtypes.PluginID(pluginID), publicKey, true)
	if err != nil {
		s.logger.WithError(err).WithFields(
			logrus.Fields{
				"public_key": publicKey,
				"plugin_id":  pluginID,
			}).Error("failed to get policies")
		return c.JSON(http.StatusInternalServerError, NewErrorResponse("failed to get policies"))
	}

	return c.JSON(http.StatusOK, policies)
}

func (s *Server) CreatePluginPolicy(c echo.Context) error {
	var policy vtypes.PluginPolicy
	if err := c.Bind(&policy); err != nil {
		return fmt.Errorf("fail to parse request, err: %w", err)
	}

	// We re-init spec as verification server doesn't have spec defined

	if err := s.spec.ValidatePluginPolicy(policy); err != nil {
		s.logger.WithError(err).Error("failed to validate spec policy")
		return c.JSON(http.StatusBadRequest, NewErrorResponse("failed to validate policy"))
	}

	if policy.ID.String() == "" {
		policy.ID = uuid.New()
	}

	if !s.verifyPolicySignature(policy) {
		s.logger.Error("invalid policy signature")
		return c.JSON(http.StatusForbidden, NewErrorResponse("Invalid policy signature"))
	}

	newPolicy, err := s.policy.CreatePolicy(c.Request().Context(), policy)
	if err != nil {
		s.logger.WithError(err).Error("Failed to create spec policy")
		return c.JSON(http.StatusInternalServerError, NewErrorResponse("failed to create policy"))
	}

	return c.JSON(http.StatusOK, newPolicy)
}

func (s *Server) UpdatePluginPolicyById(c echo.Context) error {
	var policy vtypes.PluginPolicy
	if err := c.Bind(&policy); err != nil {
		return fmt.Errorf("fail to parse request, err: %w", err)
	}

	if err := s.spec.ValidatePluginPolicy(policy); err != nil {
		s.logger.WithError(err).
			WithField("plugin_id", policy.PluginID).
			WithField("policy_id", policy.ID).
			Error("Failed to validate spec policy")
		return c.JSON(http.StatusBadRequest, NewErrorResponse("failed to validate policy"))
	}

	if !s.verifyPolicySignature(policy) {
		s.logger.Error("invalid policy signature")
		return c.JSON(http.StatusForbidden, NewErrorResponse("Invalid policy signature"))
	}

	updatedPolicy, err := s.policy.UpdatePolicy(c.Request().Context(), policy)
	if err != nil {
		s.logger.WithError(err).Error("Failed to update spec policy")
		return c.JSON(http.StatusInternalServerError, NewErrorResponse("failed to update policy"))
	}

	return c.JSON(http.StatusOK, updatedPolicy)
}

func (s *Server) DeletePluginPolicyById(c echo.Context) error {
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

func (s *Server) GetPolicySchema(c echo.Context) error {
	pluginID := c.Request().Header.Get("plugin_id") // this is a unique identifier; this won't be needed once the DCA and Payroll are separate services
	if pluginID == "" {
		return c.JSON(http.StatusBadRequest, NewErrorResponse("missing required header: plugin_id"))
	}

	// TODO: need to deal with both DCA and Payroll plugins
	keyPath := filepath.Join("spec", pluginID, "dcaPluginUiSchema.json")
	jsonData, err := os.ReadFile(keyPath)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, NewErrorResponse("failed to read spec schema"))
	}

	var data map[string]interface{}
	jsonErr := json.Unmarshal(jsonData, &data)
	if jsonErr != nil {
		s.logger.WithError(jsonErr).Error("Failed to parse spec schema")
		return c.JSON(http.StatusInternalServerError, NewErrorResponse("failed to parse spec schema"))
	}
	return c.JSON(http.StatusOK, data)
}

func (s *Server) GetRecipeSpecification(c echo.Context) error {
	recipeSpec, err := s.spec.GetRecipeSpecification()
	if err != nil {
		s.logger.WithError(err).Error("Failed to get recipe spec")
		return c.JSON(http.StatusInternalServerError, NewErrorResponse("failed to get recipe spec"))
	}
	return c.JSON(http.StatusOK, recipeSpec)
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
