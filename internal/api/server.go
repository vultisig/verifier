package api

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/vultisig/verifier/common"
	"github.com/vultisig/verifier/config"
	"github.com/vultisig/verifier/internal/service"
	"github.com/vultisig/verifier/internal/sigutil"
	"github.com/vultisig/verifier/internal/storage"
	"github.com/vultisig/verifier/internal/storage/postgres"
	"github.com/vultisig/verifier/internal/tasks"
	"github.com/vultisig/verifier/internal/types"
	vv "github.com/vultisig/verifier/internal/vultisig_validator"
	types2 "github.com/vultisig/verifier/types"
	"github.com/vultisig/verifier/vault"

	"github.com/DataDog/datadog-go/statsd"
	"github.com/go-playground/validator/v10"
	"github.com/hibiken/asynq"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
	"github.com/sirupsen/logrus"
	"github.com/vultisig/mobile-tss-lib/tss"
	"github.com/vultisig/verifier/internal/clientutil"
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
	policyService, err := service.NewPolicyService(db, client)
	if err != nil {
		logrus.Fatalf("Failed to initialize policy service: %v", err)
	}

	authService := service.NewAuthService(jwtSecret)

	return &Server{
		cfg:           cfg,
		redis:         redis,
		client:        client,
		inspector:     inspector,
		sdClient:      sdClient,
		vaultStorage:  vaultStorage,
		db:            db,
		logger:        logrus.WithField("service", "verifier-server").Logger,
		policyService: policyService,
		authService:   authService,
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

	grp := e.Group("/vault")
	grp.POST("/create", s.CreateVault)
	grp.POST("/reshare", s.ReshareVault)
	grp.GET("/get/:publicKeyECDSA", s.GetVault)     // Get Vault Data
	grp.GET("/exist/:publicKeyECDSA", s.ExistVault) // Check if Vault exists

	grp.POST("/sign", s.SignMessages)                     // Sign messages
	grp.GET("/sign/response/:taskId", s.GetKeysignResult) // Get keysign result

	pluginGroup := e.Group("/plugin", s.userAuthMiddleware)
	pluginGroup.POST("/policy", s.CreatePluginPolicy)
	pluginGroup.PUT("/policy", s.UpdatePluginPolicyById)
	pluginGroup.GET("/policy", s.GetAllPluginPolicies)
	pluginGroup.GET("/policy/:policyId", s.GetPluginPolicyById)
	pluginGroup.DELETE("/policy/:policyId", s.DeletePluginPolicyById)

	// if s.mode == "verifier" {
	// 	e.POST("/login", s.UserLogin)
	// 	e.GET("/users/me", s.GetLoggedUser, s.userAuthMiddleware)
	//
	// 	pluginsGroup := e.Group("/plugins")
	// 	pluginsGroup.GET("", s.GetPlugins)
	// 	pluginsGroup.GET("/:pluginId", s.GetPlugin)
	// 	pluginsGroup.POST("", s.CreatePlugin, s.userAuthMiddleware)
	// 	pluginsGroup.PATCH("/:pluginId", s.UpdatePlugin, s.userAuthMiddleware)
	// 	pluginsGroup.DELETE("/:pluginId", s.DeletePlugin, s.userAuthMiddleware)
	// }

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

func (s *Server) CreateVault(c echo.Context) error {
	var req types2.VaultCreateRequest
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
	if err := s.sdClient.Count("vault.create", 1, nil, 1); err != nil {
		s.logger.Errorf("fail to count metric, err: %v", err)
	}

	result, err := s.redis.Get(c.Request().Context(), req.SessionID)
	if err == nil && result != "" {
		return c.NoContent(http.StatusOK)
	}

	if err := s.redis.Set(c.Request().Context(), req.SessionID, req.SessionID, 5*time.Minute); err != nil {
		s.logger.Errorf("fail to set session, err: %v", err)
	}
	_, err = s.client.Enqueue(asynq.NewTask(tasks.TypeKeyGenerationDKLS, buf),
		asynq.MaxRetry(-1),
		asynq.Timeout(7*time.Minute),
		asynq.Retention(10*time.Minute),
		asynq.Queue(tasks.QUEUE_NAME))
	if err != nil {
		return fmt.Errorf("fail to enqueue task, err: %w", err)
	}
	return c.NoContent(http.StatusOK)
}

// ReshareVault is a handler to reshare a vault
func (s *Server) ReshareVault(c echo.Context) error {
	var req types2.ReshareRequest
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

func (s *Server) extractXPassword(c echo.Context) (string, error) {
	passwd := c.Request().Header.Get("x-password")
	if passwd == "" {
		return "", fmt.Errorf("vault backup password is required")
	}

	rawPwd, err := base64.StdEncoding.DecodeString(passwd)
	if err == nil && len(rawPwd) > 0 {
		passwd = string(rawPwd)
	} else {
		s.logger.Infof("fail to unescape password, err: %v", err)
	}

	return passwd, nil
}

func (s *Server) GetVault(c echo.Context) error {
	publicKeyECDSA := c.Param("publicKeyECDSA")
	if publicKeyECDSA == "" {
		return fmt.Errorf("public key is required")
	}
	if !s.isValidHash(publicKeyECDSA) {
		return c.NoContent(http.StatusBadRequest)
	}
	passwd, err := s.extractXPassword(c)
	if err != nil {
		return fmt.Errorf("fail to extract password, err: %w", err)
	}

	filePathName := common.GetVaultBackupFilename(publicKeyECDSA)
	content, err := s.vaultStorage.GetVault(filePathName)
	if err != nil {
		wrappedErr := fmt.Errorf("fail to read file in GetVault, err: %w", err)
		s.logger.Error(wrappedErr)
		return wrappedErr
	}

	v, err := common.DecryptVaultFromBackup(passwd, content)
	if err != nil {
		return fmt.Errorf("fail to decrypt vault from the backup, err: %w", err)
	}

	return c.JSON(http.StatusOK, types2.VaultGetResponse{
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

	var req types2.KeysignRequest
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

	filePathName := common.GetVaultBackupFilename(req.PublicKey)
	content, err := s.vaultStorage.GetVault(filePathName)
	if err != nil {
		wrappedErr := fmt.Errorf("fail to read file in SignMessages, err: %w", err)
		s.logger.Infof("fail to read file in SignMessages, err: %v", err)
		s.logger.Error(wrappedErr)
		return wrappedErr
	}
	_, err = common.DecryptVaultFromBackup(req.VaultPassword, content)
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

	filePathName := common.GetVaultBackupFilename(publicKeyECDSA)
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
		Message      string `json:"message"`
		Signature    string `json:"signature"`
		DerivePath   string `json:"derive_path"`
		ChainCodeHex string `json:"chain_code_hex"`
		PublicKey    string `json:"public_key"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request format",
		})
	}

	// Validate required fields
	if err := clientutil.ValidateAuthRequest(
		req.Message, req.Signature, req.PublicKey, req.DerivePath, req.ChainCodeHex,
	); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
	}

	// Verify the message format matches the expected format from client
	expectedMessage := clientutil.GenerateHexMessage(req.PublicKey)
	if req.Message != expectedMessage {
		s.logger.Warnf("Message mismatch: expected %s, got %s", expectedMessage, req.Message)
		// Allow some flexibility in message format by continuing anyway
	}

	msgBytes, err := hex.DecodeString(strings.TrimPrefix(req.Message, "0x"))
	if err != nil {
		s.logger.Errorf("failed to decode message: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid message format",
		})
	}

	sigBytes, err := hex.DecodeString(strings.TrimPrefix(req.Signature, "0x"))
	if err != nil {
		s.logger.Errorf("failed to decode signature: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid signature format",
		})
	}

	success, err := sigutil.VerifySignature(req.PublicKey,
		req.ChainCodeHex,
		msgBytes,
		sigBytes)
	if err != nil {
		s.logger.Errorf("signature verification failed: %v", err)
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Signature verification failed",
		})
	}
	if !success {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Invalid signature",
		})
	}

	token, err := s.authService.GenerateToken(req.PublicKey)
	if err != nil {
		s.logger.Error("failed to generate token:", err)
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to generate auth token",
		})
	}

	// Store logged-in user's public key in cache for quick access
	cacheKey := "user_pubkey:" + token[0:10]                                          // Use first part of token as a cache key
	err = s.redis.Set(c.Request().Context(), cacheKey, req.PublicKey, 7*24*time.Hour) // Same as token expiration
	if err != nil {
		s.logger.Warnf("Failed to cache user info: %v", err)
		// Continue anyway since this is not critical
	}

	return c.JSON(http.StatusOK, map[string]string{"token": token})
}

func (s *Server) RefreshToken(c echo.Context) error {
	var req struct {
		Token string `json:"token"`
	}

	if err := c.Bind(&req); err != nil {
		s.logger.Errorf("fail to decode token, err: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request format",
		})
	}

	if req.Token == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Missing token",
		})
	}

	newToken, err := s.authService.RefreshToken(req.Token)
	if err != nil {
		s.logger.Errorf("fail to refresh token, err: %v", err)
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Invalid or expired token",
		})
	}

	return c.JSON(http.StatusOK, map[string]string{"token": newToken})
}

// AuthMiddleware verifies JWT tokens for protected routes
func (s *Server) AuthMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		// Get token from header
		authHeader := c.Request().Header.Get("Authorization")

		// Extract the token using our utility
		tokenString, err := clientutil.ExtractBearerToken(authHeader)
		if err != nil {
			return c.JSON(http.StatusUnauthorized, map[string]string{
				"error": err.Error(),
			})
		}

		// Validate token
		claims, err := s.authService.ValidateToken(tokenString)
		if err != nil {
			return c.JSON(http.StatusUnauthorized, map[string]string{
				"error": "Invalid token",
			})
		}

		// Store user info in context for future use
		c.Set("user_public_key", claims.PublicKey)

		return next(c)
	}
}
