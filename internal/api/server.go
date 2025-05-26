package api

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

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
	"github.com/vultisig/verifier/internal/types"
	vv "github.com/vultisig/verifier/internal/vultisig_validator"
	tv "github.com/vultisig/verifier/types"
	"github.com/vultisig/verifier/vault"
)

type Server struct {
	cfg           config.VerifierConfig
	db            storage.DatabaseStorage
	redis         *storage.RedisStorage
	vaultStorage  vault.Storage
	client        *asynq.Client
	inspector     *asynq.Inspector
	sdClient      *statsd.Client
	policyService service.Policy
	pluginService service.Plugin
	authService   *service.AuthService
	logger        *logrus.Logger
}

// NewServer returns a new server.
func NewServer(
	cfg config.VerifierConfig,
	db *postgres.PostgresBackend,
	redis *storage.RedisStorage,
	vaultStorage vault.Storage,
	client *asynq.Client,
	inspector *asynq.Inspector,
	sdClient *statsd.Client,
	jwtSecret string,
) *Server {

	var err error

	logger := logrus.WithField("service", "verifier-server").Logger

	policyService, err := service.NewPolicyService(db, client)
	if err != nil {
		logrus.Fatalf("Failed to initialize policy service: %v", err)
	}

	pluginService, err := service.NewPluginService(db, logger)
	if err != nil {
		logrus.Fatalf("Failed to initialize plugin service: %v", err)
	}

	authService := service.NewAuthService(jwtSecret, db, logrus.WithField("service", "auth-service").Logger)

	// Start periodic cleanup of expired nonces
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			if err := db.CleanupExpiredNonces(context.Background()); err != nil {
				logger.Errorf("Failed to cleanup expired nonces: %v", err)
			}
		}
	}()

	return &Server{
		cfg:           cfg,
		redis:         redis,
		client:        client,
		inspector:     inspector,
		sdClient:      sdClient,
		vaultStorage:  vaultStorage,
		db:            db,
		logger:        logger,
		policyService: policyService,
		authService:   authService,
		pluginService: pluginService,
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
	e.POST("/auth/refresh", s.RefreshToken)

	// Token management endpoints
	tokenGroup := e.Group("/auth/tokens")
	tokenGroup.Use(s.VaultAuthMiddleware)
	tokenGroup.DELETE("/:tokenId", s.RevokeToken)
	tokenGroup.DELETE("/all", s.RevokeAllTokens)
	tokenGroup.GET("", s.GetActiveTokens)

	// Protected vault endpoints
	vaultGroup := e.Group("/vault")
	vaultGroup.POST("/reshare", s.ReshareVault)
	vaultGroup.GET("/get/:pluginId/:publicKeyECDSA", s.GetVault, s.VaultAuthMiddleware)     // Get Vault Data
	vaultGroup.GET("/exist/:pluginId/:publicKeyECDSA", s.ExistVault, s.VaultAuthMiddleware) // Check if Vault exists
	vaultGroup.GET("/sign/response/:taskId", s.GetKeysignResult, s.VaultAuthMiddleware)     // Get keysign result

	pluginGroup := e.Group("/plugin", s.userAuthMiddleware)
	pluginGroup.POST("/policy", s.CreatePluginPolicy)
	pluginGroup.PUT("/policy", s.UpdatePluginPolicyById)
	pluginGroup.POST("/sign", s.SignPluginMessages)

	pluginGroup.GET("/policies", s.GetAllPluginPolicies)
	pluginGroup.GET("/policy/:policyId", s.GetPluginPolicyById)
	pluginGroup.DELETE("/policy/:policyId", s.DeletePluginPolicyById)
	pluginGroup.GET("/policies/:policyId/history", s.GetPluginPolicyTransactionHistory, s.AuthMiddleware)

	pluginsGroup := e.Group("/plugins")
	pluginsGroup.GET("", s.GetPlugins)
	pluginsGroup.GET("/:pluginId", s.GetPlugin)
	pluginsGroup.POST("", s.CreatePlugin, s.userAuthMiddleware)
	pluginsGroup.POST("/:pluginId", s.UpdatePlugin, s.userAuthMiddleware)
	pluginsGroup.DELETE("/:pluginId", s.DeletePlugin, s.userAuthMiddleware)
	pluginsGroup.POST("/:pluginId/tags", s.AttachPluginTag, s.userAuthMiddleware)
	pluginsGroup.DELETE("/:pluginId/tags/:tagId", s.DetachPluginTag, s.userAuthMiddleware)

	pluginsGroup.GET("/:pluginId/reviews", s.GetReviews)
	pluginsGroup.POST("/:pluginId/reviews", s.CreateReview, s.AuthMiddleware)

	categoriesGroup := e.Group("/categories")
	categoriesGroup.GET("", s.GetCategories)

	tagsGroup := e.Group("/tags")
	tagsGroup.GET("", s.GetTags)

	pricingsGroup := e.Group("/pricing")
	pricingsGroup.GET("/:pricingId", s.GetPricing)
	pricingsGroup.POST("", s.CreatePricing, s.userAuthMiddleware)
	pricingsGroup.DELETE("/:pricingId", s.DeletePricing, s.userAuthMiddleware)
	syncGroup := e.Group("/sync", s.userAuthMiddleware)

	syncGroup.POST("/transaction", s.CreateTransaction)
	syncGroup.PUT("/transaction", s.UpdateTransaction)

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
	var req tv.ReshareRequest
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

// TODO: Make those handlers require jwt auth
func (s *Server) CreateTransaction(c echo.Context) error {
	var reqTx types.TransactionHistory
	if err := c.Bind(&reqTx); err != nil {
		return c.NoContent(http.StatusBadRequest)
	}

	existingTx, _ := s.db.GetTransactionByHash(c.Request().Context(), reqTx.TxHash)
	if existingTx != nil {
		if existingTx.Status != types.StatusSigningFailed &&
			existingTx.Status != types.StatusRejected {
			return c.NoContent(http.StatusConflict)
		}

		if err := s.db.UpdateTransactionStatus(c.Request().Context(), existingTx.ID, types.StatusPending, reqTx.Metadata); err != nil {
			s.logger.Errorf("fail to update transaction status: %v", err)
			return c.NoContent(http.StatusInternalServerError)
		}
		return c.NoContent(http.StatusOK)
	}

	if _, err := s.db.CreateTransactionHistory(c.Request().Context(), reqTx); err != nil {
		s.logger.Errorf("fail to create transaction, err: %v", err)
		return c.NoContent(http.StatusInternalServerError)
	}
	return c.NoContent(http.StatusOK)
}

func (s *Server) UpdateTransaction(c echo.Context) error {
	var reqTx types.TransactionHistory
	if err := c.Bind(&reqTx); err != nil {
		return c.NoContent(http.StatusBadRequest)
	}

	existingTx, _ := s.db.GetTransactionByHash(c.Request().Context(), reqTx.TxHash)
	if existingTx == nil {
		return c.NoContent(http.StatusNotFound)
	}

	if err := s.db.UpdateTransactionStatus(c.Request().Context(), existingTx.ID, reqTx.Status, reqTx.Metadata); err != nil {
		s.logger.Errorf("fail to update transaction status, err: %v", err)
		return c.NoContent(http.StatusInternalServerError)
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
		return c.JSON(http.StatusBadRequest, NewErrorResponse("Invalid request format"))
	}

	// Validate required fields
	if err := clientutil.ValidateAuthRequest(
		req.Message, req.Signature, req.PublicKey, req.ChainCodeHex,
	); err != nil {
		return c.JSON(http.StatusBadRequest, NewErrorResponse(err.Error()))
	}

	// Parse message to extract nonce and expiry
	fmt.Println("req.Message", req.Message)
	nonce, expiryTime, err := parseAuthMessage(req.Message)
	fmt.Println("nonce", nonce)
	fmt.Println("expiryTime", expiryTime)
	if err != nil {
		return c.JSON(http.StatusBadRequest, NewErrorResponse(err.Error()))
	}

	// Validate expiry time --> expiry time should be in the future [15 minutes]
	if time.Now().After(expiryTime) {
		return c.JSON(http.StatusBadRequest, NewErrorResponse("Message has expired"))
	}

	// Check if expiry is too far in the future (15 minutes)
	fmt.Println("expiryTime.Sub(time.Now())", expiryTime.Sub(time.Now()))
	fmt.Println("time.Duration(s.cfg.Auth.NonceExpiryMinutes)*time.Minute", time.Duration(s.cfg.Auth.NonceExpiryMinutes)*time.Minute)
	if expiryTime.Sub(time.Now()) >= time.Duration(s.cfg.Auth.NonceExpiryMinutes)*time.Minute {
		return c.JSON(http.StatusBadRequest, NewErrorResponse("Expiry time too far in the future"))
	}

	// // Check if nonce has been used
	// exists, err := s.db.CheckNonceExists(c.Request().Context(), nonce)
	// if err != nil {
	// 	s.logger.Errorf("Failed to check nonce: %v", err)
	// 	return c.JSON(http.StatusInternalServerError, NewErrorResponse("Failed to validate nonce"))
	// }
	// if exists {
	// 	return c.JSON(http.StatusBadRequest, NewErrorResponse("Nonce already used"))
	// }

	// Decode signature from hex (remove 0x prefix first)
	sigBytes, err := hex.DecodeString(req.Signature)
	if err != nil {
		return c.JSON(http.StatusBadRequest, NewErrorResponse("Invalid signature format"))
	}

	ethAddress, _, _, err := address.GetAddress(req.PublicKey, req.ChainCodeHex, common.Ethereum)
	if err != nil {
		s.logger.Errorf("failed to get derived public key: %v", err)
		return c.JSON(http.StatusBadRequest, NewErrorResponse("Invalid public key format"))
	}
	fmt.Println("ethAddress", ethAddress)

	//extract the public key from the signature , make sure it match the eth public key
	success, err := sigutil.VerifyEthAddressSignature(ecommon.HexToAddress(ethAddress), []byte(req.Message), sigBytes)
	if err != nil {
		s.logger.Errorf("signature verification failed: %v", err)
		return c.JSON(http.StatusUnauthorized, NewErrorResponse("Signature verification failed: "+err.Error()))
	}
	fmt.Println("success", success)

	if !success {
		return c.JSON(http.StatusUnauthorized, NewErrorResponse("Invalid signature"))
	}

	// Store the nonce
	// if err := s.db.StoreNonce(c.Request().Context(), nonce, req.PublicKey); err != nil {
	// 	s.logger.Errorf("Failed to store nonce: %v", err)
	// 	return c.JSON(http.StatusInternalServerError, NewErrorResponse("Failed to store nonce"))
	// }

	// Generate JWT token with the public key
	token, err := s.authService.GenerateToken(c.Request().Context(), req.PublicKey)
	if err != nil {
		s.logger.Error("failed to generate token:", err)
		return c.JSON(http.StatusInternalServerError, NewErrorResponse("Failed to generate auth token"))
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
	lines := strings.Split(strings.TrimSpace(message), "\n")
	if len(lines) < 4 {
		return "", time.Time{}, fmt.Errorf("invalid message format")
	}

	var nonce string
	var expiryStr string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Nonce: ") {
			nonce = strings.TrimPrefix(line, "Nonce: ")
		} else if strings.HasPrefix(line, "Expires At: ") {
			expiryStr = strings.TrimPrefix(line, "Expires At: ")
		}
	}

	if nonce == "" || expiryStr == "" {
		return "", time.Time{}, fmt.Errorf("missing nonce or expiry time")
	}

	expiryTime, err := time.Parse(time.RFC3339, expiryStr)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("invalid expiry time format: %w", err)
	}

	return nonce, expiryTime, nil
}

func (s *Server) RefreshToken(c echo.Context) error {
	var req struct {
		Token string `json:"token"`
	}

	if err := c.Bind(&req); err != nil {
		s.logger.Errorf("fail to decode token, err: %v", err)
		return c.JSON(http.StatusBadRequest, NewErrorResponse("Invalid request format"))
	}

	if req.Token == "" {
		return c.JSON(http.StatusBadRequest, NewErrorResponse("Missing token"))
	}

	newToken, err := s.authService.RefreshToken(c.Request().Context(), req.Token)
	if err != nil {
		s.logger.Errorf("fail to refresh token, err: %v", err)
		return c.JSON(http.StatusUnauthorized, NewErrorResponse("Invalid or expired token"))
	}

	return c.JSON(http.StatusOK, map[string]string{"token": newToken})
}

// RevokeToken revokes a specific token
func (s *Server) RevokeToken(c echo.Context) error {
	tokenID := c.Param("tokenId")
	if tokenID == "" {
		return c.JSON(http.StatusBadRequest, NewErrorResponse("Missing token ID"))
	}

	vaultKey, ok := c.Get("vault_public_key").(string)
	if !ok {
		return c.JSON(http.StatusUnauthorized, NewErrorResponse("Unauthorized"))
	}

	err := s.authService.RevokeToken(c.Request().Context(), vaultKey, tokenID)
	if err != nil {
		s.logger.Errorf("Failed to revoke token: %v", err)
		switch {
		case errors.Is(err, service.ErrTokenNotFound):
			return c.JSON(http.StatusNotFound, NewErrorResponse("Token not found"))
		case errors.Is(err, service.ErrNotOwner):
			return c.JSON(http.StatusForbidden, NewErrorResponse("Unauthorized token revocation"))
		case errors.Is(err, service.ErrBeginTx):
			return c.JSON(http.StatusInternalServerError, NewErrorResponse("Failed to begin transaction"))
		case errors.Is(err, service.ErrGetToken):
			return c.JSON(http.StatusInternalServerError, NewErrorResponse("Failed to get token"))
		case errors.Is(err, service.ErrRevokeToken):
			return c.JSON(http.StatusInternalServerError, NewErrorResponse("Failed to revoke token"))
		case errors.Is(err, service.ErrCommitTx):
			return c.JSON(http.StatusInternalServerError, NewErrorResponse("Failed to commit transaction"))
		default:
			return c.JSON(http.StatusInternalServerError, NewErrorResponse("Failed to revoke token"))
		}
	}

	return c.NoContent(http.StatusOK)
}

// RevokeAllTokens revokes all tokens for the authenticated vault
func (s *Server) RevokeAllTokens(c echo.Context) error {
	// Get public key from context (set by VaultAuthMiddleware)
	publicKey, ok := c.Get("vault_public_key").(string)
	if !ok {
		return c.JSON(http.StatusInternalServerError, NewErrorResponse("Failed to get vault public key"))
	}

	err := s.authService.RevokeAllTokens(c.Request().Context(), publicKey)
	if err != nil {
		s.logger.Errorf("Failed to revoke all tokens: %v", err)
		return c.JSON(http.StatusInternalServerError, NewErrorResponse("Failed to revoke all tokens"))
	}

	return c.NoContent(http.StatusOK)
}

// GetActiveTokens returns all active tokens for the authenticated vault
func (s *Server) GetActiveTokens(c echo.Context) error {
	// Get public key from context (set by VaultAuthMiddleware)
	publicKey, ok := c.Get("vault_public_key").(string)
	if !ok {
		return c.JSON(http.StatusInternalServerError, NewErrorResponse("Failed to get vault public key"))
	}

	tokens, err := s.authService.GetActiveTokens(c.Request().Context(), publicKey)
	if err != nil {
		s.logger.Errorf("Failed to get active tokens: %v", err)
		return c.JSON(http.StatusInternalServerError, NewErrorResponse("Failed to get active tokens"))
	}

	return c.JSON(http.StatusOK, tokens)
}
