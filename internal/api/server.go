package api

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/sirupsen/logrus"

	ecommon "github.com/ethereum/go-ethereum/common"

	vv "github.com/vultisig/verifier/common/vultisig_validator"
	"github.com/vultisig/verifier/config"
	"github.com/vultisig/verifier/internal/clientutil"
	"github.com/vultisig/verifier/internal/logging"
	internalMetrics "github.com/vultisig/verifier/internal/metrics"
	"github.com/vultisig/verifier/internal/service"
	"github.com/vultisig/verifier/internal/sigutil"
	"github.com/vultisig/verifier/internal/storage"
	"github.com/vultisig/verifier/internal/storage/postgres"
	"github.com/vultisig/verifier/internal/syncer"
	"github.com/vultisig/verifier/plugin/tasks"
	"github.com/vultisig/verifier/plugin/tx_indexer"
	vtypes "github.com/vultisig/verifier/types"
	"github.com/vultisig/verifier/vault"
	"github.com/vultisig/vultisig-go/address"
	"github.com/vultisig/vultisig-go/common"
	vgtypes "github.com/vultisig/vultisig-go/types"
)

type Server struct {
	cfg              config.VerifierConfig
	db               storage.DatabaseStorage
	redis            *storage.RedisStorage
	vaultStorage     vault.Storage
	asynqClient      *asynq.Client
	inspector        *asynq.Inspector
	policyService    service.Policy
	pluginService    service.Plugin
	feeService       service.Fees
	authService      *service.AuthService
	txIndexerService *tx_indexer.Service
	httpMetrics      *internalMetrics.HTTPMetrics
	logger           *logrus.Logger
}

// NewServer returns a new server.
func NewServer(
	cfg config.VerifierConfig,
	db *postgres.PostgresBackend,
	redis *storage.RedisStorage,
	vaultStorage vault.Storage,
	asynqClient *asynq.Client,
	inspector *asynq.Inspector,
	jwtSecret string,
	txIndexerService *tx_indexer.Service,
	httpMetrics *internalMetrics.HTTPMetrics,
) *Server {

	var err error

	logger := logrus.WithField("service", "verifier-server").Logger

	// Create syncer for synchronous policy sync
	syncer := syncer.NewPolicySyncer(db)

	policyService, err := service.NewPolicyService(db, syncer)
	if err != nil {
		logrus.Fatalf("Failed to initialize policy service: %v", err)
	}

	pluginService, err := service.NewPluginService(db, redis, logger)
	if err != nil {
		logrus.Fatalf("Failed to initialize plugin service: %v", err)
	}

	feeService, err := service.NewFeeService(db, asynqClient, logger, cfg.Fees)
	if err != nil {
		logrus.Fatalf("Failed to initialize fee service: %v", err)
	}

	authService := service.NewAuthService(jwtSecret, db, logrus.WithField("service", "auth-service").Logger)

	return &Server{
		cfg:              cfg,
		redis:            redis,
		asynqClient:      asynqClient,
		inspector:        inspector,
		vaultStorage:     vaultStorage,
		db:               db,
		logger:           logger,
		policyService:    policyService,
		authService:      authService,
		pluginService:    pluginService,
		feeService:       feeService,
		txIndexerService: txIndexerService,
		httpMetrics:      httpMetrics,
	}
}

func (s *Server) StartServer() error {
	e := echo.New()

	// Add HTTP metrics middleware if enabled
	if s.httpMetrics != nil {
		e.Use(s.httpMetrics.Middleware())
	}

	e.Use(logging.LoggerMiddleware(logrus.StandardLogger()))
	e.Use(middleware.Recover())
	e.Use(middleware.BodyLimit("2M")) // set maximum allowed size for a request body to 2M
	e.Use(middleware.CORS())
	limiterStore := middleware.NewRateLimiterMemoryStoreWithConfig(
		middleware.RateLimiterMemoryStoreConfig{Rate: 5, Burst: 30, ExpiresIn: 5 * time.Minute},
	)
	e.Use(middleware.RateLimiter(limiterStore))

	e.Validator = &vv.VultisigValidator{Validator: validator.New()}

	e.GET("/healthz", s.Ping)

	// Auth endpoints - not requiring authentication
	e.POST("/auth", s.Auth)
	e.POST("/auth/refresh", s.RefreshToken, s.VaultAuthMiddleware) // only when user has logged in with their vault

	// Token management endpoints
	tokenGroup := e.Group("/auth/tokens", s.VaultAuthMiddleware)
	tokenGroup.DELETE("/:tokenId", s.RevokeToken)
	tokenGroup.DELETE("/all", s.RevokeAllTokens)
	tokenGroup.GET("", s.GetActiveTokens)

	// Protected vault endpoints
	vaultGroup := e.Group("/vault", s.VaultAuthMiddleware)
	// Reshare vault endpoint, only user who already log in can request resharing
	vaultGroup.POST("/reshare", s.ReshareVault)
	vaultGroup.GET("/get/:pluginId/:publicKeyECDSA", s.GetVault)     // Get Vault Data
	vaultGroup.GET("/exist/:pluginId/:publicKeyECDSA", s.ExistVault) // Check if Vault exists

	// Sign endpoint, plugin should authenticate themselves using the API Key issued by the Verifier
	pluginSigner := e.Group("/plugin-signer", s.PluginAuthMiddleware)
	pluginSigner.POST("/sign", s.SignPluginMessages)               // Sign messages
	pluginSigner.GET("/sign/response/:taskId", s.GetKeysignResult) // Get keysign result

	pluginGroup := e.Group("/plugin", s.VaultAuthMiddleware)
	pluginGroup.DELETE("/:pluginId", s.DeletePlugin) // Delete plugin
	pluginGroup.POST("/policy", s.CreatePluginPolicy)
	pluginGroup.PUT("/policy", s.UpdatePluginPolicyById)
	pluginGroup.GET("/policies/:pluginId", s.GetAllPluginPolicies)
	pluginGroup.GET("/policy/:policyId", s.GetPluginPolicyById)
	pluginGroup.GET("/policy/:pluginId/total-count", s.GetPluginInstallationsCountByID)
	pluginGroup.DELETE("/policy/:policyId", s.DeletePluginPolicyById)
	pluginGroup.GET("/policies/:policyId/history", s.GetPluginPolicyTransactionHistory)
	pluginGroup.GET("/transactions", s.GetPluginTransactionHistory)

	// fee group. These should only be accessible by the plugin server
	feeGroup := e.Group("/fees", s.PluginAuthMiddleware)
	feeGroup.GET("/publickey/:publicKey", s.GetPublicKeyFees)
	feeGroup.POST("/collected", s.MarkCollected)

	// user fee group. These should only be accessible by the plugin server
	userFeeGroup := e.Group("/fee", s.VaultAuthMiddleware)
	userFeeGroup.GET("/status", s.GetUserFees)
	userFeeGroup.GET("/plugins/:pluginId/transactions", s.GetPluginFeeHistory)

	pluginsGroup := e.Group("/plugins")
	pluginsGroup.GET("", s.GetPlugins)
	pluginsGroup.GET("/:pluginId", s.GetPlugin)
	pluginsGroup.GET("/installed", s.GetInstalledPlugins, s.VaultAuthMiddleware)

	pluginsGroup.GET("/:pluginId/reviews", s.GetReviews)
	pluginsGroup.POST("/:pluginId/reviews", s.CreateReview, s.AuthMiddleware)
	pluginsGroup.GET("/:pluginId/recipe-specification", s.GetPluginRecipeSpecification)
	pluginsGroup.GET("/:pluginId/recipe-functions", s.GetPluginRecipeFunctions)
	pluginsGroup.POST("/:pluginId/recipe-specification/suggest", s.GetPluginRecipeSpecificationSuggest)
	pluginsGroup.GET("/:pluginId/average-rating", s.GetPluginAvgRating)

	categoriesGroup := e.Group("/categories")
	categoriesGroup.GET("", s.GetCategories)

	tagsGroup := e.Group("/tags")
	tagsGroup.GET("", s.GetTags)

	pricingsGroup := e.Group("/pricing")
	pricingsGroup.GET("/:pricingId", s.GetPricing)

	return e.Start(fmt.Sprintf(":%d", s.cfg.Server.Port))
}

func (s *Server) Ping(c echo.Context) error {
	return c.String(http.StatusOK, "Verifier server is running")
}

// ReshareVault is a handler to reshare a vault
func (s *Server) ReshareVault(c echo.Context) error {
	s.logger.Info("ReshareVault: Starting reshare vault request")

	var req vtypes.ReshareRequest
	if err := c.Bind(&req); err != nil {
		s.logger.WithError(err).Error("ReshareVault: Failed to parse request body")
		return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage(msgRequestParseFailed))
	}

	if err := req.IsValid(); err != nil {
		s.logger.WithError(err).Error("ReshareVault: Request validation failed")
		return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage(msgRequestValidationFailed))
	}

	// Check if session exists in Redis
	result, err := s.redis.Get(c.Request().Context(), req.SessionID)
	if err == nil && result != "" {
		s.logger.WithField("session_id", req.SessionID).Info("Session already active, skipping enqueue")
		status := http.StatusOK
		return c.JSON(status, NewSuccessResponse(status, "already_exists"))
	}

	// First, notify plugin server synchronously
	ctx, cancel := context.WithTimeout(c.Request().Context(), 30*time.Second)
	defer cancel()

	if err := s.notifyPluginServerReshare(ctx, req); err != nil {
		s.logger.WithError(err).Error("ReshareVault: Plugin server notification failed")
		return c.JSON(http.StatusServiceUnavailable, NewErrorResponseWithMessage(msgPluginServerUnavailable))
	}

	// Store session in Redis
	if err := s.redis.Set(c.Request().Context(), req.SessionID, req.SessionID, 5*time.Minute); err != nil {
		s.logger.WithError(err).Error("ReshareVault: Failed to store session in Redis")
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgStoreSessionFailed))
	}

	// Enqueue background task
	buf, err := json.Marshal(req)
	if err != nil {
		s.logger.WithError(err).Error("ReshareVault: Failed to marshal request")
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgRequestProcessFailed))
	}

	_, err = s.asynqClient.Enqueue(asynq.NewTask(tasks.TypeReshareDKLS, buf),
		asynq.MaxRetry(-1),
		asynq.Timeout(7*time.Minute),
		asynq.Retention(10*time.Minute),
		asynq.Queue(tasks.QUEUE_NAME))
	if err != nil {
		s.logger.WithError(err).Error("ReshareVault: Failed to enqueue task")
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgReshareQueueFailed))
	}

	status := http.StatusOK
	return c.JSON(status, NewSuccessResponse(status, "queued"))
}

// notifyPluginServerReshare sends the reshare request to the plugin server
func (s *Server) notifyPluginServerReshare(ctx context.Context, req vtypes.ReshareRequest) error {
	// Look up plugin server endpoint
	plugin, err := s.db.FindPluginById(ctx, nil, vtypes.PluginID(req.PluginID))
	if err != nil {
		return fmt.Errorf("failed to find plugin: %w", err)
	}
	keyInfo, err := s.db.GetAPIKeyByPluginId(ctx, req.PluginID)
	if err != nil {
		return fmt.Errorf("failed to get api key by id: %w", err)
	}

	// Prepare and send request to plugin server
	pluginURL := fmt.Sprintf("%s/vault/reshare", plugin.ServerEndpoint)
	payload, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", pluginURL, bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+keyInfo.ApiKey)
	client := &http.Client{Timeout: 30 * time.Second}

	resp, err := client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to call plugin server: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			s.logger.WithError(err).Error("Failed to close response body")
		}
	}()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		s.logger.Errorf("Plugin server error (status %d): %s", resp.StatusCode, string(body))
		return fmt.Errorf("plugin server returned status %d", resp.StatusCode)
	}

	return nil
}

func (s *Server) GetVault(c echo.Context) error {
	publicKeyECDSA := c.Param("publicKeyECDSA")
	if publicKeyECDSA == "" {
		return c.JSON(http.StatusBadRequest, errors.New(msgRequiredPublicKey))
	}
	if !s.isValidHash(publicKeyECDSA) {
		return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage(msgInvalidPublicKey))
	}
	pluginId := c.Param("pluginId")
	if pluginId == "" {
		return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage(msgRequiredPluginID))
	}
	filePathName := common.GetVaultBackupFilename(publicKeyECDSA, pluginId)
	content, err := s.vaultStorage.GetVault(filePathName)
	if err != nil {
		wrappedErr := fmt.Errorf("fail to read file in GetVault, err: %w", err)
		s.logger.Error(wrappedErr)
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(wrappedErr.Error()))
	}

	v, err := common.DecryptVaultFromBackup(s.cfg.EncryptionSecret, content)
	if err != nil {
		wrappedErr := fmt.Errorf("fail to decrypt vault from the backup, err: %w", err)
		s.logger.Error(wrappedErr)
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(wrappedErr.Error()))
	}

	return c.JSON(http.StatusOK, NewSuccessResponse(http.StatusOK, vgtypes.VaultGetResponse{
		Name:           v.Name,
		PublicKeyEcdsa: v.PublicKeyEcdsa,
		PublicKeyEddsa: v.PublicKeyEddsa,
		HexChainCode:   v.HexChainCode,
		LocalPartyId:   v.LocalPartyId,
	}))
}

// GetKeysignResult is a handler to get the keysign response
func (s *Server) GetKeysignResult(c echo.Context) error {
	taskID := c.Param("taskId")
	if taskID == "" {
		return errors.New(msgRequiredTaskID)
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
		return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage(msgRequiredPublicKey))
	}
	if !s.isValidHash(publicKeyECDSA) {
		return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage(msgInvalidPublicKey))
	}
	pluginId := c.Param("pluginId")
	if pluginId == "" {
		return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage(msgRequiredPluginID))
	}

	filePathName := common.GetVaultBackupFilename(publicKeyECDSA, pluginId)
	exist, err := s.vaultStorage.Exist(filePathName)
	if err != nil {
		s.logger.WithError(err).Error("failed to check vault existence")
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgInternalError))
	}
	if !exist {
		return c.JSON(http.StatusNotFound, NewErrorResponseWithMessage(msgVaultNotFound))
	}

	status := http.StatusOK
	return c.JSON(status, NewSuccessResponse(status, "ok"))
}

func (s *Server) Auth(c echo.Context) error {
	var req struct {
		Message      string `json:"message"`        // hex encoded message
		Signature    string `json:"signature"`      // hex encoded signature
		ChainCodeHex string `json:"chain_code_hex"` // hex encoded chain code
		PublicKey    string `json:"public_key"`     // hex encoded public key
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage(msgInvalidRequestFormat))
	}

	// Validate required fields
	if err := clientutil.ValidateAuthRequest(
		req.Message, req.Signature, req.PublicKey, req.ChainCodeHex,
	); err != nil {
		return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage(msgInvalidRequestFormat))
	}

	// Parse message to extract nonce and expiry
	nonce, expiryTime, err := parseAuthMessage(req.Message)
	if err != nil {
		return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage(msgInvalidMessageFormat))
	}

	// Validate expiry time
	if time.Now().After(expiryTime) {
		return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage(msgMessageExpired))
	}

	// Decode signature from hex (remove 0x prefix first)
	sigBytes, err := hex.DecodeString(strings.TrimPrefix(req.Signature, "0x"))
	if err != nil {
		s.logger.WithError(err).Error("Failed to decode signature")
		return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage(msgInvalidSignatureFormat))
	}

	ethAddress, _, _, err := address.GetAddress(req.PublicKey, req.ChainCodeHex, common.Ethereum)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get address")
		return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage(msgInvalidPublicKeyFormat))
	}

	// extract the public key from the signature, make sure it matches the eth public key
	success, err := sigutil.VerifyEthAddressSignature(ecommon.HexToAddress(ethAddress), []byte(req.Message), sigBytes)
	if err != nil {
		s.logger.WithError(err).Error("signature verification failed")
		return c.JSON(http.StatusUnauthorized, NewErrorResponseWithMessage(msgSignatureVerificationFailed))
	}
	if !success {
		return c.JSON(http.StatusUnauthorized, NewErrorResponseWithMessage(msgInvalidSignature))
	}

	// Unique nonce-public key identifier
	nonceKey := fmt.Sprintf("%s:%s", req.PublicKey, nonce)

	// Check if expiry is too far in the future
	if time.Until(expiryTime) > time.Duration(s.cfg.Auth.NonceExpiryMinutes)*time.Minute {
		// We should still store the nonce in redis to avoid delayed replays
		if err := s.redis.Set(c.Request().Context(), nonceKey, "1", time.Until(expiryTime)); err != nil {
			s.logger.WithError(err).Error("Failed to store nonce")
			return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgNonceStoreFailed))
		}
		return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage(msgExpiryTooFarInFuture))
	}

	// Check if nonce has been used in Redis
	exists, err := s.redis.Exists(c.Request().Context(), nonceKey)
	if err != nil {
		s.logger.WithError(err).Errorf("Nonce already used")
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgNonceUsed))
	}
	if exists {
		return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage(msgNonceUsed))
	}

	// Store the nonce in Redis with expiry
	if err := s.redis.Set(c.Request().Context(), nonceKey, "1", time.Until(expiryTime)); err != nil {
		s.logger.WithError(err).Errorf("Failed to store nonce")
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgNonceStoreFailed))
	}

	// Generate JWT token with the public key
	token, err := s.authService.GenerateToken(c.Request().Context(), req.PublicKey)
	if err != nil {
		s.logger.Error("failed to generate token:", err)
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgTokenGenerateFailed))
	}

	// Store logged-in user's public key in cache for quick access
	cacheKey := "user_pubkey:" + token
	err = s.redis.Set(c.Request().Context(), cacheKey, req.PublicKey, 7*24*time.Hour) // Same as token expiration
	if err != nil {
		s.logger.WithError(err).Warnf("Failed to cache user info")
	}

	err = s.db.WithTransaction(c.Request().Context(), func(ctx context.Context, tx pgx.Tx) error {
		_, _, err := s.db.IsTrialActive(ctx, tx, req.PublicKey)
		return err
	})
	if err != nil {
		s.logger.WithError(err).Warnf("Failed to check trial info")
	}

	status := http.StatusOK
	resp := map[string]string{"token": token}
	return c.JSON(status, NewSuccessResponse(status, resp))
}

// parseAuthMessage extracts nonce and expiry time from the auth message
func parseAuthMessage(message string) (string, time.Time, error) {
	var authData struct {
		Message   string `json:"message"`
		Nonce     string `json:"nonce"`
		ExpiresAt string `json:"expiresAt"`
		Address   string `json:"address"`
	}

	if err := json.Unmarshal([]byte(message), &authData); err != nil {
		return "", time.Time{}, fmt.Errorf("invalid message format: %w", err)
	}

	if authData.Nonce == "" || authData.ExpiresAt == "" {
		return "", time.Time{}, errors.New(msgNonceOrExpiryRequired)
	}

	expiryTime, err := time.Parse(time.RFC3339, authData.ExpiresAt)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("invalid expiry time format: %w", err)
	}

	return authData.Nonce, expiryTime, nil
}

func (s *Server) RefreshToken(c echo.Context) error {
	var req struct {
		Token string `json:"token"`
	}

	if err := c.Bind(&req); err != nil {
		s.logger.WithError(err).Errorf("fail to decode token")
		return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage(msgInvalidRequestFormat))
	}

	if req.Token == "" {
		return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage(msgMissingTokenID))
	}

	newToken, err := s.authService.RefreshToken(c.Request().Context(), req.Token)
	if err != nil {
		s.logger.WithError(err).Error("fail to refresh token")
		return c.JSON(http.StatusUnauthorized, NewErrorResponseWithMessage(msgInvalidOrExpiredToken))
	}

	status := http.StatusOK
	resp := map[string]string{"token": newToken}
	return c.JSON(status, NewSuccessResponse(status, resp))
}

// RevokeToken revokes a specific token
func (s *Server) RevokeToken(c echo.Context) error {
	tokenID := c.Param("tokenId")
	if tokenID == "" {
		return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage(msgMissingTokenID))
	}

	vaultKey, ok := c.Get("vault_public_key").(string)
	if !ok || vaultKey == "" {
		s.logger.Warn("Missing vault_public_key in context")
		return c.JSON(http.StatusUnauthorized, NewErrorResponseWithMessage(msgUnauthorized))
	}

	err := s.authService.RevokeToken(c.Request().Context(), vaultKey, tokenID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to revoke token")
		switch {
		case errors.Is(err, service.ErrTokenNotFound):
			return c.JSON(http.StatusNotFound, NewErrorResponseWithMessage(msgTokenNotFound))
		case errors.Is(err, service.ErrNotOwner):
			return c.JSON(http.StatusForbidden, NewErrorResponseWithMessage(msgUnauthorizedTokenRevocation))
		case errors.Is(err, service.ErrBeginTx):
			return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgTransactionBeginFailed))
		case errors.Is(err, service.ErrGetToken):
			return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgTokenGetFailed))
		case errors.Is(err, service.ErrRevokeToken):
			return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgTokenRevokeFailed))
		case errors.Is(err, service.ErrCommitTx):
			return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgTransactionCommitFailed))
		default:
			return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgTokenRevokeFailed))
		}
	}

	status := http.StatusOK
	return c.JSON(status, NewSuccessResponse(status, "revoked"))
}

// RevokeAllTokens revokes all tokens for the authenticated vault
func (s *Server) RevokeAllTokens(c echo.Context) error {
	// Get public key from context (set by VaultAuthMiddleware)
	publicKey, ok := c.Get("vault_public_key").(string)
	if !ok || publicKey == "" {
		s.logger.Warn("Missing vault_public_key in context")
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgVaultPublicKeyGetFailed))
	}

	err := s.authService.RevokeAllTokens(c.Request().Context(), publicKey)
	if err != nil {
		s.logger.WithError(err).Errorf("Failed to revoke all tokens")
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgRevokeAllTokensFailed))
	}

	status := http.StatusOK
	return c.JSON(status, NewSuccessResponse(status, "revoked"))
}

// GetActiveTokens returns all active tokens for the authenticated vault
func (s *Server) GetActiveTokens(c echo.Context) error {
	// Get public key from context (set by VaultAuthMiddleware)
	publicKey, ok := c.Get("vault_public_key").(string)
	if !ok || publicKey == "" {
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgVaultPublicKeyGetFailed))
	}

	tokens, err := s.authService.GetActiveTokens(c.Request().Context(), publicKey)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get active tokens")
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgGetActiveTokensFailed))
	}

	return c.JSON(http.StatusOK, NewSuccessResponse(http.StatusOK, tokens))
}

// notifyPluginServerDeletePlugin user would like to delete a plugin, we need to notify the plugin server
func (s *Server) notifyPluginServerDeletePlugin(ctx context.Context, id vtypes.PluginID, publicKeyEcdsa string) error {
	// Look up plugin server endpoint
	plugin, err := s.db.FindPluginById(ctx, nil, id)
	if err != nil {
		return fmt.Errorf("failed to find plugin: %w", err)
	}
	keyInfo, err := s.db.GetAPIKeyByPluginId(ctx, id.String())
	if err != nil {
		return fmt.Errorf("failed to get api key by id: %w", err)
	}

	// Prepare and send request to plugin server
	pluginURL := fmt.Sprintf("%s/vault/%s/%s", strings.TrimRight(plugin.ServerEndpoint, "/"), id, publicKeyEcdsa)
	httpReq, err := http.NewRequestWithContext(ctx, "DELETE", pluginURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+keyInfo.ApiKey)
	client := &http.Client{Timeout: 30 * time.Second}

	resp, err := client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to call plugin server: %w", err)
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			s.logger.WithError(closeErr).Errorln("Failed to close response body")
		}
	}()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		s.logger.WithError(fmt.Errorf("status %d: %s", resp.StatusCode, string(body))).Error("notifyPluginServerDeletePlugin: Plugin server error")
		return fmt.Errorf("plugin server returned status %d", resp.StatusCode)
	}

	return nil
}

// DeletePlugin deletes a plugin and its associated policies and vault
func (s *Server) DeletePlugin(c echo.Context) error {
	pluginID := c.Param("pluginId")
	if pluginID == "" {
		return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage(msgRequiredPluginID))
	}
	// Get public key from context (set by VaultAuthMiddleware)
	publicKey, ok := c.Get("vault_public_key").(string)
	if !ok || publicKey == "" {
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgVaultPublicKeyGetFailed))
	}
	if pluginID == vtypes.PluginVultisigFees_feee.String() {
		return c.JSON(http.StatusForbidden, NewErrorResponseWithMessage("Unable to uninstall due to outstanding fees"))
	}

	if err := s.notifyPluginServerDeletePlugin(c.Request().Context(), vtypes.PluginID(pluginID), publicKey); err != nil {
		s.logger.WithError(err).Errorf("Failed to notify plugin server for deletion of plugin %s", pluginID)
		return c.JSON(http.StatusServiceUnavailable, NewErrorResponseWithMessage(msgPluginServerUnavailable))
	}
	// remove plugin policies
	if err := s.policyService.DeleteAllPolicies(c.Request().Context(), vtypes.PluginID(pluginID), publicKey); err != nil {
		s.logger.Errorf("Failed to delete plugin policies: %v", err)
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgPoliciesDeleteFailed))
	}
	fileName := common.GetVaultBackupFilename(publicKey, pluginID)
	// delete the vault
	if err := s.vaultStorage.DeleteFile(fileName); err != nil {
		s.logger.WithError(err).Error("Failed to delete vault file")
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgVaultShareDeleteFailed))
	}

	status := http.StatusOK
	return c.JSON(status, NewSuccessResponse(status, "deleted"))
}
