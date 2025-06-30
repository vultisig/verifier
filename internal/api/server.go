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

	"github.com/vultisig/verifier/tx_indexer"

	"github.com/DataDog/datadog-go/statsd"
	"github.com/go-playground/validator/v10"
	"github.com/hibiken/asynq"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	"github.com/sirupsen/logrus"
	"github.com/vultisig/mobile-tss-lib/tss"

	ecommon "github.com/ethereum/go-ethereum/common"

	"github.com/vultisig/verifier/address"
	"github.com/vultisig/verifier/common"
	"github.com/vultisig/verifier/config"
	"github.com/vultisig/verifier/internal/clientutil"
	"github.com/vultisig/verifier/internal/service"
	"github.com/vultisig/verifier/internal/sigutil"
	"github.com/vultisig/verifier/internal/storage"
	"github.com/vultisig/verifier/internal/storage/postgres"
	"github.com/vultisig/verifier/internal/tasks"
	vv "github.com/vultisig/verifier/internal/vultisig_validator"
	tv "github.com/vultisig/verifier/types"
	"github.com/vultisig/verifier/vault"
)

type Server struct {
	cfg              config.VerifierConfig
	db               storage.DatabaseStorage
	redis            *storage.RedisStorage
	vaultStorage     vault.Storage
	asynqClient      *asynq.Client
	inspector        *asynq.Inspector
	sdClient         *statsd.Client
	policyService    service.Policy
	pluginService    service.Plugin
	feeService       service.Fees
	authService      *service.AuthService
	txIndexerService *tx_indexer.Service
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
	sdClient *statsd.Client,
	jwtSecret string,
	txIndexerService *tx_indexer.Service,
) *Server {

	var err error

	logger := logrus.WithField("service", "verifier-server").Logger

	policyService, err := service.NewPolicyService(db, asynqClient)
	if err != nil {
		logrus.Fatalf("Failed to initialize policy service: %v", err)
	}

	pluginService, err := service.NewPluginService(db, redis, logger)
	if err != nil {
		logrus.Fatalf("Failed to initialize plugin service: %v", err)
	}

	feeService, err := service.NewFeeService(db, asynqClient, logger)
	if err != nil {
		logrus.Fatalf("Failed to initialize fee service: %v", err)
	}

	authService := service.NewAuthService(jwtSecret, db, logrus.WithField("service", "auth-service").Logger)

	return &Server{
		cfg:              cfg,
		redis:            redis,
		asynqClient:      asynqClient,
		inspector:        inspector,
		sdClient:         sdClient,
		vaultStorage:     vaultStorage,
		db:               db,
		logger:           logger,
		policyService:    policyService,
		authService:      authService,
		pluginService:    pluginService,
		feeService:       feeService,
		txIndexerService: txIndexerService,
	}
}

func (s *Server) StartServer() error {
	e := echo.New()
	e.Logger.SetLevel(log.DEBUG)
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.BodyLimit("2M")) // set maximum allowed size for a request body to 2M
	e.Use(s.statsdMiddleware)
	e.Use(middleware.CORS())
	limiterStore := middleware.NewRateLimiterMemoryStoreWithConfig(
		middleware.RateLimiterMemoryStoreConfig{Rate: 5, Burst: 30, ExpiresIn: 5 * time.Minute},
	)
	e.Use(middleware.RateLimiter(limiterStore))

	e.Validator = &vv.VultisigValidator{Validator: validator.New()}

	e.GET("/ping", s.Ping)
	e.GET("/getDerivedPublicKey", s.GetDerivedPublicKey)

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
	vaultGroup.POST("/sign", s.SignPluginMessages, s.PluginAuthMiddleware)               // Sign messages
	vaultGroup.GET("/sign/response/:taskId", s.GetKeysignResult, s.PluginAuthMiddleware) // Get keysign result

	pluginGroup := e.Group("/plugin", s.VaultAuthMiddleware)
	pluginGroup.DELETE("/:pluginId", s.DeletePlugin) // Delete plugin
	pluginGroup.POST("/policy", s.CreatePluginPolicy)
	pluginGroup.PUT("/policy", s.UpdatePluginPolicyById)
	pluginGroup.GET("/policies", s.GetAllPluginPolicies)
	pluginGroup.GET("/policy/:policyId", s.GetPluginPolicyById)
	pluginGroup.DELETE("/policy/:policyId", s.DeletePluginPolicyById)
	pluginGroup.GET("/policies/:policyId/history", s.GetPluginPolicyTransactionHistory)

	// fee group. These should only be accessible by the plugin server
	feeGroup := e.Group("/fees")
	feeGroup.GET("/policy/:policyId", s.GetPluginPolicyFees, s.FeeAuthMiddleware)
	feeGroup.GET("/plugin/:pluginId", s.GetPluginFees, s.FeeAuthMiddleware)
	feeGroup.GET("/publickey/:publicKey", s.GetPublicKeyFees, s.FeeAuthMiddleware)

	/* placeholder for future plugin endpoints
	pluginGroup.GET("/fees", func(c echo.Context) error {
		s.client.Enqueue(asynq.NewTask(tasks.TypeRecurringFeeRecord, []byte{}, asynq.Queue(tasks.QUEUE_NAME)))
		return c.NoContent(http.StatusOK)
	}) */

	pluginsGroup := e.Group("/plugins")
	pluginsGroup.GET("", s.GetPlugins)
	pluginsGroup.GET("/:pluginId", s.GetPlugin)

	pluginsGroup.GET("/:pluginId/reviews", s.GetReviews)
	pluginsGroup.POST("/:pluginId/reviews", s.CreateReview, s.AuthMiddleware)
	pluginsGroup.GET("/:pluginId/recipe-specification", s.GetPluginRecipeSpecification)

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

// GetDerivedPublicKey is a handler to get the derived public key
func (s *Server) GetDerivedPublicKey(c echo.Context) error {
	publicKey := c.QueryParam("publicKey")
	if publicKey == "" {
		return fmt.Errorf("publicKey is required")
	}
	hexChainCode := c.QueryParam("hexChainCode")
	if hexChainCode == "" {
		return fmt.Errorf("hexChainCode is required")
	}
	derivePath := c.QueryParam("derivePath")
	if derivePath == "" {
		return fmt.Errorf("derivePath is required")
	}
	isEdDSA := false
	isEdDSAstr := c.QueryParam("isEdDSA")
	if isEdDSAstr == "true" {
		isEdDSA = true
	}

	derivedPublicKey, err := tss.GetDerivedPubKey(publicKey, hexChainCode, derivePath, isEdDSA)
	if err != nil {
		return fmt.Errorf("fail to get derived public key from tss, err: %w", err)
	}

	return c.JSON(http.StatusOK, derivedPublicKey)
}

// ReshareVault is a handler to reshare a vault
func (s *Server) ReshareVault(c echo.Context) error {
	s.logger.Info("ReshareVault: Starting reshare vault request")

	var req tv.ReshareRequest
	if err := c.Bind(&req); err != nil {
		s.logger.Errorf("ReshareVault: Failed to parse request body: %v", err)
		return fmt.Errorf("fail to parse request, err: %w", err)
	}

	if err := req.IsValid(); err != nil {
		s.logger.Errorf("ReshareVault: Request validation failed: %v", err)
		return fmt.Errorf("invalid request, err: %w", err)
	}

	// Check if session exists in Redis
	result, err := s.redis.Get(c.Request().Context(), req.SessionID)
	if err == nil && result != "" {
		return c.NoContent(http.StatusOK)
	}

	// First, notify plugin server synchronously
	ctx, cancel := context.WithTimeout(c.Request().Context(), 30*time.Second)
	defer cancel()

	if err := s.notifyPluginServerReshare(ctx, req); err != nil {
		s.logger.Errorf("ReshareVault: Plugin server notification failed: %v", err)
		return c.JSON(http.StatusServiceUnavailable, NewErrorResponse(http.StatusServiceUnavailable, "Plugin server is currently unavailable", ""))
	}

	// Store session in Redis
	if err := s.redis.Set(c.Request().Context(), req.SessionID, req.SessionID, 5*time.Minute); err != nil {
		s.logger.Errorf("ReshareVault: Failed to store session in Redis: %v", err)
		return c.JSON(http.StatusInternalServerError, NewErrorResponse(http.StatusInternalServerError, "Failed to store session", err.Error()))
	}

	// Enqueue background task
	buf, err := json.Marshal(req)
	if err != nil {
		s.logger.Errorf("ReshareVault: Failed to marshal request: %v", err)
		return c.JSON(http.StatusInternalServerError, NewErrorResponse(http.StatusInternalServerError, "Failed to process request", err.Error()))
	}

	_, err = s.asynqClient.Enqueue(asynq.NewTask(tasks.TypeReshareDKLS, buf),
		asynq.MaxRetry(-1),
		asynq.Timeout(7*time.Minute),
		asynq.Retention(10*time.Minute),
		asynq.Queue(tasks.QUEUE_NAME))
	if err != nil {
		s.logger.Errorf("ReshareVault: Failed to enqueue task: %v", err)
		return c.JSON(http.StatusInternalServerError, NewErrorResponse(http.StatusInternalServerError, "Failed to queue reshare task", err.Error()))
	}

	return c.NoContent(http.StatusOK)
}

// notifyPluginServerReshare sends the reshare request to the plugin server
func (s *Server) notifyPluginServerReshare(ctx context.Context, req tv.ReshareRequest) error {
	// Look up plugin server endpoint
	plugin, err := s.db.FindPluginById(ctx, nil, tv.PluginID(req.PluginID))
	if err != nil {
		return fmt.Errorf("failed to find plugin: %w", err)
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
	client := &http.Client{Timeout: 30 * time.Second}

	resp, err := client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to call plugin server: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			s.logger.Errorf("notifyPluginServerReshare: Failed to close response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		s.logger.Errorf("notifyPluginServerReshare: Plugin server error (status %d): %s", resp.StatusCode, string(body))
		return fmt.Errorf("plugin server returned status %d", resp.StatusCode)
	}

	return nil
}

func (s *Server) GetVault(c echo.Context) error {
	publicKeyECDSA := c.Param("publicKeyECDSA")
	if publicKeyECDSA == "" {
		return fmt.Errorf("public key is required")
	}
	if !s.isValidHash(publicKeyECDSA) {
		return c.NoContent(http.StatusBadRequest)
	}
	pluginId := c.Param("pluginId")
	if pluginId == "" {
		return fmt.Errorf("plugin id is required")
	}
	filePathName := common.GetVaultBackupFilename(publicKeyECDSA, pluginId)
	content, err := s.vaultStorage.GetVault(filePathName)
	if err != nil {
		wrappedErr := fmt.Errorf("fail to read file in GetVault, err: %w", err)
		s.logger.Error(wrappedErr)
		return wrappedErr
	}

	v, err := common.DecryptVaultFromBackup(s.cfg.EncryptionSecret, content)
	if err != nil {
		return fmt.Errorf("fail to decrypt vault from the backup, err: %w", err)
	}

	return c.JSON(http.StatusOK, tv.VaultGetResponse{
		Name:           v.Name,
		PublicKeyEcdsa: v.PublicKeyEcdsa,
		PublicKeyEddsa: v.PublicKeyEddsa,
		HexChainCode:   v.HexChainCode,
		LocalPartyId:   v.LocalPartyId,
	})
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
		return fmt.Errorf("public key is required")
	}
	if !s.isValidHash(publicKeyECDSA) {
		return c.NoContent(http.StatusBadRequest)
	}
	pluginId := c.Param("pluginId")
	if pluginId == "" {
		return fmt.Errorf("plugin id is required")
	}

	filePathName := common.GetVaultBackupFilename(publicKeyECDSA, pluginId)
	exist, err := s.vaultStorage.Exist(filePathName)
	if err != nil || !exist {
		return c.NoContent(http.StatusBadRequest)
	}
	return c.NoContent(http.StatusOK)
}

func (s *Server) Auth(c echo.Context) error {
	var req struct {
		Message      string `json:"message"`        // hex encoded message
		Signature    string `json:"signature"`      // hex encoded signature
		ChainCodeHex string `json:"chain_code_hex"` // hex encoded chain code
		PublicKey    string `json:"public_key"`     // hex encoded public key
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, NewErrorResponse(http.StatusBadRequest, "Invalid request format", err.Error()))
	}

	// Validate required fields
	if err := clientutil.ValidateAuthRequest(
		req.Message, req.Signature, req.PublicKey, req.ChainCodeHex,
	); err != nil {
		return c.JSON(http.StatusBadRequest, NewErrorResponse(http.StatusBadRequest, err.Error(), err.Error()))
	}

	// Parse message to extract nonce and expiry
	nonce, expiryTime, err := parseAuthMessage(req.Message)
	if err != nil {
		return c.JSON(http.StatusBadRequest, NewErrorResponse(http.StatusBadRequest, err.Error(), err.Error()))
	}

	// Validate expiry time
	if time.Now().After(expiryTime) {
		return c.JSON(http.StatusBadRequest, NewErrorResponse(http.StatusBadRequest, "Message has expired", ""))
	}

	// Decode signature from hex (remove 0x prefix first)
	sigBytes, err := hex.DecodeString(strings.TrimPrefix(req.Signature, "0x"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, NewErrorResponse(http.StatusBadRequest, "Invalid signature format", err.Error()))
	}

	ethAddress, _, _, err := address.GetAddress(req.PublicKey, req.ChainCodeHex, common.Ethereum)
	if err != nil {
		s.logger.Errorf("failed to get derived public key: %v", err)
		return c.JSON(http.StatusBadRequest, NewErrorResponse(http.StatusBadRequest, "Invalid public key format", err.Error()))
	}

	// extract the public key from the signature , make sure it match the eth public key
	success, err := sigutil.VerifyEthAddressSignature(ecommon.HexToAddress(ethAddress), []byte(req.Message), sigBytes)
	if err != nil {
		s.logger.Errorf("signature verification failed: %v", err)
		return c.JSON(http.StatusUnauthorized, NewErrorResponse(http.StatusUnauthorized, "Signature verification failed: "+err.Error(), err.Error()))
	}

	if !success {
		return c.JSON(http.StatusUnauthorized, NewErrorResponse(http.StatusUnauthorized, "Invalid signature", ""))
	}

	// Unique nonce-public key identifier
	nonceKey := fmt.Sprintf("%s:%s", req.PublicKey, nonce)

	// Check if expiry is too far in the future
	if time.Until(expiryTime) > time.Duration(s.cfg.Auth.NonceExpiryMinutes)*time.Minute {
		// We should still store the nonce in redis to avoid delayed replays
		if err := s.redis.Set(c.Request().Context(), nonceKey, "1", time.Until(expiryTime)); err != nil {
			s.logger.Errorf("Failed to store nonce: %v", err)
			return c.JSON(http.StatusInternalServerError, NewErrorResponse(http.StatusInternalServerError, "Failed to store nonce", err.Error()))
		}
		return c.JSON(http.StatusBadRequest, NewErrorResponse(http.StatusBadRequest, "Expiry time too far in the future", ""))
	}

	// Check if nonce has been used using Redis
	exists, err := s.redis.Exists(c.Request().Context(), nonceKey)
	if err != nil {
		s.logger.Errorf("Nonce already used: %v", err)
		return c.JSON(http.StatusInternalServerError, NewErrorResponse(http.StatusInternalServerError, "Nonce was already used", err.Error()))
	}
	if exists {
		return c.JSON(http.StatusBadRequest, NewErrorResponse(http.StatusBadRequest, "Nonce already used", ""))
	}

	// Store the nonce in Redis with expiry
	if err := s.redis.Set(c.Request().Context(), nonceKey, "1", time.Until(expiryTime)); err != nil {
		s.logger.Errorf("Failed to store nonce: %v", err)
		return c.JSON(http.StatusInternalServerError, NewErrorResponse(http.StatusInternalServerError, "Failed to store nonce", err.Error()))
	}

	// Generate JWT token with the public key
	token, err := s.authService.GenerateToken(c.Request().Context(), req.PublicKey)
	if err != nil {
		s.logger.Error("failed to generate token:", err)
		return c.JSON(http.StatusInternalServerError, NewErrorResponse(http.StatusInternalServerError, "Failed to generate auth token", err.Error()))
	}

	// Store logged-in user's public key in cache for quick access
	cacheKey := "user_pubkey:" + token
	err = s.redis.Set(c.Request().Context(), cacheKey, req.PublicKey, 7*24*time.Hour) // Same as token expiration
	if err != nil {
		s.logger.Warnf("Failed to cache user info: %v", err)
		// Continue anyway since this is not critical
	}

	return c.JSON(http.StatusOK, map[string]string{"token": token})
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
		return "", time.Time{}, fmt.Errorf("missing nonce or expiry time")
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
		s.logger.Errorf("fail to decode token, err: %v", err)
		return c.JSON(http.StatusBadRequest, NewErrorResponse(http.StatusBadRequest, "Invalid request format", err.Error()))
	}

	if req.Token == "" {
		return c.JSON(http.StatusBadRequest, NewErrorResponse(http.StatusBadRequest, "Missing token", ""))
	}

	newToken, err := s.authService.RefreshToken(c.Request().Context(), req.Token)
	if err != nil {
		s.logger.Errorf("fail to refresh token, err: %v", err)
		return c.JSON(http.StatusUnauthorized, NewErrorResponse(http.StatusUnauthorized, "Invalid or expired token", err.Error()))
	}

	return c.JSON(http.StatusOK, map[string]string{"token": newToken})
}

// RevokeToken revokes a specific token
func (s *Server) RevokeToken(c echo.Context) error {
	tokenID := c.Param("tokenId")
	if tokenID == "" {
		return c.JSON(http.StatusBadRequest, NewErrorResponse(http.StatusBadRequest, "Missing token ID", ""))
	}

	vaultKey, ok := c.Get("vault_public_key").(string)
	if !ok {
		return c.JSON(http.StatusUnauthorized, NewErrorResponse(http.StatusUnauthorized, "Unauthorized", ""))
	}

	err := s.authService.RevokeToken(c.Request().Context(), vaultKey, tokenID)
	if err != nil {
		s.logger.Errorf("Failed to revoke token: %v", err)
		switch {
		case errors.Is(err, service.ErrTokenNotFound):
			return c.JSON(http.StatusNotFound, NewErrorResponse(http.StatusNotFound, "Token not found", ""))
		case errors.Is(err, service.ErrNotOwner):
			return c.JSON(http.StatusForbidden, NewErrorResponse(http.StatusForbidden, "Unauthorized token revocation", ""))
		case errors.Is(err, service.ErrBeginTx):
			return c.JSON(http.StatusInternalServerError, NewErrorResponse(http.StatusInternalServerError, "Failed to begin transaction", err.Error()))
		case errors.Is(err, service.ErrGetToken):
			return c.JSON(http.StatusInternalServerError, NewErrorResponse(http.StatusInternalServerError, "Failed to get token", err.Error()))
		case errors.Is(err, service.ErrRevokeToken):
			return c.JSON(http.StatusInternalServerError, NewErrorResponse(http.StatusInternalServerError, "Failed to revoke token", err.Error()))
		case errors.Is(err, service.ErrCommitTx):
			return c.JSON(http.StatusInternalServerError, NewErrorResponse(http.StatusInternalServerError, "Failed to commit transaction", err.Error()))
		default:
			return c.JSON(http.StatusInternalServerError, NewErrorResponse(http.StatusInternalServerError, "Failed to revoke token", err.Error()))
		}
	}

	return c.NoContent(http.StatusOK)
}

// RevokeAllTokens revokes all tokens for the authenticated vault
func (s *Server) RevokeAllTokens(c echo.Context) error {
	// Get public key from context (set by VaultAuthMiddleware)
	publicKey, ok := c.Get("vault_public_key").(string)
	if !ok {
		return c.JSON(http.StatusInternalServerError, NewErrorResponse(http.StatusInternalServerError, "Failed to get vault public key", ""))
	}

	err := s.authService.RevokeAllTokens(c.Request().Context(), publicKey)
	if err != nil {
		s.logger.Errorf("Failed to revoke all tokens: %v", err)
		return c.JSON(http.StatusInternalServerError, NewErrorResponse(http.StatusInternalServerError, "Failed to revoke all tokens", err.Error()))
	}

	return c.NoContent(http.StatusOK)
}

// GetActiveTokens returns all active tokens for the authenticated vault
func (s *Server) GetActiveTokens(c echo.Context) error {
	// Get public key from context (set by VaultAuthMiddleware)
	publicKey, ok := c.Get("vault_public_key").(string)
	if !ok {
		return c.JSON(http.StatusInternalServerError, NewErrorResponse(http.StatusInternalServerError, "Failed to get vault public key", ""))
	}

	tokens, err := s.authService.GetActiveTokens(c.Request().Context(), publicKey)
	if err != nil {
		s.logger.Errorf("Failed to get active tokens: %v", err)
		return c.JSON(http.StatusInternalServerError, NewErrorResponse(http.StatusInternalServerError, "Failed to get active tokens", err.Error()))
	}

	return c.JSON(http.StatusOK, tokens)
}

// notifyPluginServerDeletePlugin user would like to delete a plugin, we need to notify the plugin server
func (s *Server) notifyPluginServerDeletePlugin(ctx context.Context, id tv.PluginID, publicKeyEcdsa string) error {
	// Look up plugin server endpoint
	plugin, err := s.db.FindPluginById(ctx, nil, id)
	if err != nil {
		return fmt.Errorf("failed to find plugin: %w", err)
	}

	// Prepare and send request to plugin server
	pluginURL := fmt.Sprintf("%s/vault/%s/%s", strings.TrimRight(plugin.ServerEndpoint, "/"), id, publicKeyEcdsa)
	httpReq, err := http.NewRequestWithContext(ctx, "DELETE", pluginURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
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
		s.logger.Errorf("notifyPluginServerDeletePlugin: Plugin server error (status %d): %s", resp.StatusCode, string(body))
		return fmt.Errorf("plugin server returned status %d", resp.StatusCode)
	}

	return nil
}

// DeletePlugin deletes a plugin and its associated policies and vault
func (s *Server) DeletePlugin(c echo.Context) error {
	pluginID := c.Param("pluginId")
	if pluginID == "" {
		return c.JSON(http.StatusBadRequest, NewErrorResponse(http.StatusBadRequest, "Plugin ID is required", ""))
	}
	// Get public key from context (set by VaultAuthMiddleware)
	publicKey, ok := c.Get("vault_public_key").(string)
	if !ok {
		return c.JSON(http.StatusInternalServerError, NewErrorResponse(http.StatusInternalServerError, "Failed to get vault public key", ""))
	}
	if err := s.notifyPluginServerDeletePlugin(c.Request().Context(), tv.PluginID(pluginID), publicKey); err != nil {
		s.logger.WithError(err).Errorf("Failed to notify plugin server for deletion of plugin %s", pluginID)
		return c.JSON(http.StatusServiceUnavailable, NewErrorResponse(http.StatusServiceUnavailable, "Plugin server is currently unavailable", ""))
	}
	// remove plugin policies
	if err := s.policyService.DeleteAllPolicies(c.Request().Context(), tv.PluginID(pluginID), publicKey); err != nil {
		s.logger.Errorf("Failed to delete plugin policies: %v", err)
		return c.JSON(http.StatusInternalServerError, NewErrorResponse(http.StatusInternalServerError, "Failed to delete plugin policies", err.Error()))
	}
	fileName := common.GetVaultBackupFilename(publicKey, pluginID)
	// delete the vault
	if err := s.vaultStorage.DeleteFile(fileName); err != nil {
		return c.JSON(http.StatusInternalServerError, NewErrorResponse(http.StatusInternalServerError, "Failed to delete vault share", err.Error()))
	}
	return c.NoContent(http.StatusOK)
}
