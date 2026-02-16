package portal

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/sirupsen/logrus"

	"github.com/vultisig/verifier/config"
	"github.com/vultisig/verifier/internal/sigutil"
	"github.com/vultisig/verifier/internal/storage"
	"github.com/vultisig/verifier/internal/storage/postgres"
	"github.com/vultisig/verifier/internal/storage/postgres/queries"
	itypes "github.com/vultisig/verifier/internal/types"
	"github.com/vultisig/vultisig-go/address"
	vcommon "github.com/vultisig/vultisig-go/common"
)

type Server struct {
	cfg           config.PortalConfig
	pool          *pgxpool.Pool
	queries       *queries.Queries
	logger        *logrus.Logger
	authService   *PortalAuthService
	inviteService *InviteService
	db            *postgres.PostgresBackend
	assetStorage  storage.PluginAssetStorage
}

func NewServer(cfg config.PortalConfig, pool *pgxpool.Pool, db *postgres.PostgresBackend, assetStorage storage.PluginAssetStorage) *Server {
	logger := logrus.WithField("service", "portal").Logger
	return &Server{
		cfg:           cfg,
		pool:          pool,
		queries:       queries.New(pool),
		logger:        logger,
		authService:   NewPortalAuthService(cfg.Server.JWTSecret, logger),
		inviteService: NewInviteService(cfg.Server.HMACSecret, cfg.Server.BaseURL),
		db:            db,
		assetStorage:  assetStorage,
	}
}

func (s *Server) Start() error {
	e := echo.New()
	e.HideBanner = true

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	s.registerRoutes(e)

	addr := fmt.Sprintf("%s:%d", s.cfg.Server.Host, s.cfg.Server.Port)
	s.logger.Infof("Starting portal server on %s", addr)
	return e.Start(addr)
}

func (s *Server) registerRoutes(e *echo.Echo) {
	e.GET("/healthz", s.Healthz)

	// Auth endpoint (public)
	e.POST("/auth", s.Auth)

	// Public plugin routes (pricings only)
	e.GET("/plugins/:id/pricings", s.GetPluginPricings)

	// Public invite validation endpoint (validates magic link, returns invite info)
	e.GET("/invite/validate", s.ValidateInvite)

	// Protected routes (require JWT auth)
	protected := e.Group("")
	protected.Use(s.JWTAuthMiddleware)
	// Plugin routes - only return plugins owned by the authenticated user
	protected.GET("/plugins", s.ListPlugins)
	protected.GET("/plugins/:id", s.GetPlugin)
	protected.GET("/plugins/:id/my-role", s.GetMyPluginRole)
	protected.PUT("/plugins/:id", s.UpdatePlugin)
	// API key management
	protected.GET("/plugins/:id/api-keys", s.GetPluginApiKeys)
	protected.POST("/plugins/:id/api-keys", s.CreatePluginApiKey)
	protected.PUT("/plugins/:id/api-keys/:keyId", s.UpdatePluginApiKey)
	protected.DELETE("/plugins/:id/api-keys/:keyId", s.DeletePluginApiKey)
	// Team management
	protected.GET("/plugins/:id/team", s.ListTeamMembers)
	protected.POST("/plugins/:id/team/invite", s.CreateInvite)
	protected.POST("/plugins/:id/team/accept", s.AcceptInvite)
	protected.DELETE("/plugins/:id/team/:publicKey", s.RemoveTeamMember)
	// Kill switch management (staff only)
	protected.GET("/plugins/:id/kill-switch", s.GetKillSwitch)
	protected.PUT("/plugins/:id/kill-switch", s.SetKillSwitch)
	// Earnings
	protected.GET("/earnings", s.GetEarnings)
	protected.GET("/earnings/summary", s.GetEarningsSummary)
	// Image management
	protected.GET("/plugins/:id/images", s.ListPluginImages)
	protected.POST("/plugins/:id/images/upload-url", s.GetImageUploadURL)
	protected.POST("/plugins/:id/images/:imageId/confirm", s.ConfirmImageUpload)
	protected.PATCH("/plugins/:id/images/:imageId", s.UpdatePluginImage)
	protected.DELETE("/plugins/:id/images/:imageId", s.DeletePluginImage)
	protected.PUT("/plugins/:id/images/order", s.ReorderPluginImages)
}

func (s *Server) Healthz(c echo.Context) error {
	return c.String(http.StatusOK, "OK")
}

// JWTAuthMiddleware validates JWT tokens and extracts user information
func (s *Server) JWTAuthMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		authHeader := c.Request().Header.Get(echo.HeaderAuthorization)
		if authHeader == "" {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "authorization header required"})
		}

		tokenParts := strings.Fields(authHeader)
		if len(tokenParts) != 2 || !strings.EqualFold(tokenParts[0], "Bearer") {
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid authorization header format"})
		}
		tokenStr := tokenParts[1]

		claims, err := s.authService.ValidateToken(tokenStr)
		if err != nil {
			s.logger.Warnf("failed to validate token: %v", err)
			return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid or expired token"})
		}

		// Store claims in context for use by handlers
		c.Set("public_key", claims.PublicKey)
		c.Set("address", claims.Address)

		return next(c)
	}
}

// AuthRequest represents the request body for authentication
type AuthRequest struct {
	Message      string `json:"message"`
	Signature    string `json:"signature"`
	PublicKey    string `json:"public_key"`
	ChainCodeHex string `json:"chain_code_hex"`
}

// AuthResponse represents the response for authentication
type AuthResponse struct {
	Token   string `json:"token"`
	Address string `json:"address"`
}

// Auth handles user authentication and returns a JWT token
func (s *Server) Auth(c echo.Context) error {
	var req AuthRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request format"})
	}

	// Validate required fields
	if req.Message == "" || req.Signature == "" || req.PublicKey == "" || req.ChainCodeHex == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "missing required fields"})
	}

	// Parse and validate the message
	var msgData struct {
		Message   string `json:"message"`
		Nonce     string `json:"nonce"`
		ExpiresAt string `json:"expiresAt"`
		Address   string `json:"address"`
	}
	if err := json.Unmarshal([]byte(req.Message), &msgData); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid message format"})
	}

	// Validate expiry time
	expiryTime, err := time.Parse(time.RFC3339, msgData.ExpiresAt)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid expiry time format"})
	}
	if time.Now().After(expiryTime) {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "message has expired"})
	}

	// Decode signature from hex
	sigBytes, err := hex.DecodeString(strings.TrimPrefix(req.Signature, "0x"))
	if err != nil {
		s.logger.WithError(err).Error("failed to decode signature")
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid signature format"})
	}

	// Get Ethereum address from public key and chain code
	ethAddress, _, _, err := address.GetAddress(req.PublicKey, req.ChainCodeHex, vcommon.Ethereum)
	if err != nil {
		s.logger.WithError(err).Error("failed to derive address")
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid public key or chain code"})
	}

	// Verify signature
	valid, err := sigutil.VerifyEthAddressSignature(common.HexToAddress(ethAddress), []byte(req.Message), sigBytes)
	if err != nil {
		s.logger.WithError(err).Error("signature verification failed")
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "signature verification failed"})
	}
	if !valid {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid signature"})
	}

	// Generate JWT token
	token, err := s.authService.GenerateToken(req.PublicKey, ethAddress)
	if err != nil {
		s.logger.WithError(err).Error("failed to generate token")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to generate token"})
	}

	s.logger.WithFields(logrus.Fields{
		"address":    ethAddress,
		"public_key": req.PublicKey,
	}).Info("user authenticated successfully")

	return c.JSON(http.StatusOK, AuthResponse{
		Token:   token,
		Address: ethAddress,
	})
}

func (s *Server) GetPlugin(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "plugin id is required"})
	}

	// Get address from JWT context (set by JWTAuthMiddleware)
	address, ok := c.Get("address").(string)
	if !ok || address == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "authentication required"})
	}

	// Only return the plugin if the authenticated user owns it
	plugin, err := s.queries.GetPluginByIDAndOwner(c.Request().Context(), &queries.GetPluginByIDAndOwnerParams{
		ID:        id,
		PublicKey: address,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "plugin not found"})
		}
		s.logger.WithError(err).Error("failed to get plugin")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}

	return c.JSON(http.StatusOK, plugin)
}

// MyRoleResponse represents the user's role for a plugin
type MyRoleResponse struct {
	Role    string `json:"role"`
	CanEdit bool   `json:"canEdit"`
}

// GetMyPluginRole returns the current user's role for a plugin
func (s *Server) GetMyPluginRole(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "plugin id is required"})
	}

	// Get address from JWT context
	address, ok := c.Get("address").(string)
	if !ok || address == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "authentication required"})
	}

	// Get the user's role for this plugin
	owner, err := s.queries.GetPluginOwnerWithRole(c.Request().Context(), &queries.GetPluginOwnerWithRoleParams{
		PluginID:  id,
		PublicKey: address,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "not a member of this plugin"})
		}
		s.logger.WithError(err).Error("failed to get plugin role")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}

	// Determine if user can edit (viewers cannot edit)
	canEdit := owner.Role != queries.PluginOwnerRoleViewer

	return c.JSON(http.StatusOK, MyRoleResponse{
		Role:    string(owner.Role),
		CanEdit: canEdit,
	})
}

func (s *Server) ListPlugins(c echo.Context) error {
	// Get address from JWT context (set by JWTAuthMiddleware)
	address, ok := c.Get("address").(string)
	if !ok || address == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "authentication required"})
	}

	// Only return plugins owned by the authenticated user
	plugins, err := s.queries.ListPluginsByOwner(c.Request().Context(), address)
	if err != nil {
		s.logger.WithError(err).Error("failed to list plugins")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}

	return c.JSON(http.StatusOK, plugins)
}

// PluginPricingResponse is the API response for plugin pricing
type PluginPricingResponse struct {
	ID        string          `json:"id"`
	PluginID  string          `json:"pluginId"`
	Type      string          `json:"type"`
	Frequency *string         `json:"frequency"`
	Amount    string          `json:"amount"`
	FeeAsset  itypes.FeeAsset `json:"fee_asset"`
	Metric    string          `json:"metric"`
}

func (s *Server) GetPluginPricings(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "plugin id is required"})
	}

	pricings, err := s.queries.GetPluginPricings(c.Request().Context(), id)
	if err != nil {
		s.logger.WithError(err).Error("failed to get plugin pricings")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}

	// Convert to API response format
	response := make([]PluginPricingResponse, len(pricings))
	for i, p := range pricings {
		var freq *string
		if p.Frequency.Valid {
			f := string(p.Frequency.PricingFrequency)
			freq = &f
		}
		response[i] = PluginPricingResponse{
			ID:        p.ID.String(),
			PluginID:  string(p.PluginID),
			Type:      string(p.Type),
			Frequency: freq,
			Amount:    strconv.FormatInt(p.Amount, 10),
			FeeAsset:  itypes.DefaultFeeAsset,
			Metric:    string(p.Metric),
		}
	}

	return c.JSON(http.StatusOK, response)
}

// PluginApiKeyResponse is the API response for plugin API keys
type PluginApiKeyResponse struct {
	ID        string  `json:"id"`
	PluginID  string  `json:"pluginId"`
	ApiKey    string  `json:"apikey"`
	CreatedAt string  `json:"createdAt"`
	ExpiresAt *string `json:"expiresAt"`
	Status    int32   `json:"status"`
}

func (s *Server) GetPluginApiKeys(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "plugin id is required"})
	}

	// Get address from JWT context (set by JWTAuthMiddleware)
	address, ok := c.Get("address").(string)
	if !ok || address == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "authentication required"})
	}

	// Verify the requester is an admin of this plugin
	owner, err := s.queries.GetPluginOwnerWithRole(c.Request().Context(), &queries.GetPluginOwnerWithRoleParams{
		PluginID:  id,
		PublicKey: address,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return c.JSON(http.StatusForbidden, map[string]string{"error": "not authorized to view API keys for this plugin"})
		}
		s.logger.WithError(err).Error("failed to check plugin ownership")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}

	// Only admins can view/manage API keys
	if owner.Role != queries.PluginOwnerRoleAdmin {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "only admins can manage API keys"})
	}

	apiKeys, err := s.queries.GetPluginApiKeys(c.Request().Context(), id)
	if err != nil {
		s.logger.WithError(err).Error("failed to get plugin api keys")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}

	// Convert to API response format
	response := make([]PluginApiKeyResponse, len(apiKeys))
	for i, k := range apiKeys {
		var expiresAt *string
		if k.ExpiresAt.Valid {
			t := k.ExpiresAt.Time.Format(time.RFC3339)
			expiresAt = &t
		}
		response[i] = PluginApiKeyResponse{
			ID:        k.ID.String(),
			PluginID:  string(k.PluginID),
			ApiKey:    maskApiKey(k.Apikey),
			CreatedAt: k.CreatedAt.Time.Format(time.RFC3339),
			ExpiresAt: expiresAt,
			Status:    k.Status,
		}
	}

	return c.JSON(http.StatusOK, response)
}

// maskApiKey masks an API key showing only the prefix and first/last 4 hex chars
// e.g., "vbt_abc123...def456"
func maskApiKey(key string) string {
	// Key format: vbt_<64 hex chars>
	if len(key) < 12 {
		return key
	}
	// Get the prefix (vbt_) and the hex part
	if strings.HasPrefix(key, "vbt_") {
		hexPart := key[4:]
		if len(hexPart) >= 8 {
			return "vbt_" + hexPart[:4] + "..." + hexPart[len(hexPart)-4:]
		}
	}
	// Fallback for keys without prefix
	return key[:4] + "..." + key[len(key)-4:]
}

// maskPayoutAddress masks a payout address showing first 6 and last 4 chars
func maskPayoutAddress(address string) string {
	if address == "" {
		return ""
	}
	if len(address) == 42 && strings.HasPrefix(address, "0x") {
		return address[:6] + "..." + address[len(address)-4:]
	}
	return address
}

// generateApiKey generates a random API key with vbt_ prefix and 32 bytes hex
func generateApiKey() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return "vbt_" + hex.EncodeToString(bytes), nil
}

// CreateApiKeyRequest represents the request to create a new API key
type CreateApiKeyRequest struct {
	ExpiresAt *string `json:"expiresAt"` // RFC3339 format, optional
}

// CreateApiKeyResponse represents the response with the full API key (shown only once)
type CreateApiKeyResponse struct {
	ID        string  `json:"id"`
	PluginID  string  `json:"pluginId"`
	ApiKey    string  `json:"apikey"` // Full key, shown only on creation
	CreatedAt string  `json:"createdAt"`
	ExpiresAt *string `json:"expiresAt"`
	Status    int32   `json:"status"`
}

// CreatePluginApiKey creates a new API key for a plugin
func (s *Server) CreatePluginApiKey(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "plugin id is required"})
	}

	// Get address from JWT context
	address, ok := c.Get("address").(string)
	if !ok || address == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "authentication required"})
	}

	// Verify the requester is an admin of this plugin
	owner, err := s.queries.GetPluginOwnerWithRole(c.Request().Context(), &queries.GetPluginOwnerWithRoleParams{
		PluginID:  id,
		PublicKey: address,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return c.JSON(http.StatusForbidden, map[string]string{"error": "not authorized to manage API keys for this plugin"})
		}
		s.logger.WithError(err).Error("failed to check plugin ownership")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}

	// Only admins can manage API keys
	if owner.Role != queries.PluginOwnerRoleAdmin {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "only admins can manage API keys"})
	}

	var req CreateApiKeyRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	// Generate the API key
	apiKey, err := generateApiKey()
	if err != nil {
		s.logger.WithError(err).Error("failed to generate API key")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to generate API key"})
	}

	// Parse optional expiry time
	var expiresAt pgtype.Timestamptz
	if req.ExpiresAt != nil && *req.ExpiresAt != "" {
		t, err := time.Parse(time.RFC3339, *req.ExpiresAt)
		if err != nil {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid expiry time format"})
		}
		expiresAt = pgtype.Timestamptz{Time: t, Valid: true}
	}

	// Create the API key in a transaction with advisory lock to prevent race conditions
	ctx := c.Request().Context()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		s.logger.WithError(err).Error("failed to begin transaction")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create API key"})
	}
	defer tx.Rollback(ctx)

	q := queries.New(tx)

	err = q.AcquireApiKeyLock(ctx, id)
	if err != nil {
		s.logger.WithError(err).Error("failed to acquire lock")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create API key"})
	}

	count, err := q.CountActiveApiKeys(ctx, id)
	if err != nil {
		s.logger.WithError(err).Error("failed to count API keys")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create API key"})
	}

	if count >= int64(s.cfg.MaxApiKeysPerPlugin) {
		return c.JSON(http.StatusConflict, map[string]string{"error": fmt.Sprintf("maximum number of API keys (%d) reached for this plugin", s.cfg.MaxApiKeysPerPlugin)})
	}

	created, err := q.CreatePluginApiKey(ctx, &queries.CreatePluginApiKeyParams{
		PluginID:  id,
		Apikey:    apiKey,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		s.logger.WithError(err).Error("failed to create API key")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create API key"})
	}

	err = tx.Commit(ctx)
	if err != nil {
		s.logger.WithError(err).Error("failed to commit transaction")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to create API key"})
	}

	var expiresAtStr *string
	if created.ExpiresAt.Valid {
		t := created.ExpiresAt.Time.Format(time.RFC3339)
		expiresAtStr = &t
	}

	s.logger.WithFields(logrus.Fields{
		"plugin_id": id,
		"key_id":    created.ID.String(),
	}).Info("API key created successfully")

	// Return the full API key (only shown once)
	return c.JSON(http.StatusCreated, CreateApiKeyResponse{
		ID:        created.ID.String(),
		PluginID:  string(created.PluginID),
		ApiKey:    apiKey, // Full key returned only on creation
		CreatedAt: created.CreatedAt.Time.Format(time.RFC3339),
		ExpiresAt: expiresAtStr,
		Status:    created.Status,
	})
}

// UpdateApiKeyRequest represents the request to update an API key
type UpdateApiKeyRequest struct {
	Status int32 `json:"status"` // 0 = disabled, 1 = enabled
}

// UpdatePluginApiKey updates an API key's status (enable/disable)
func (s *Server) UpdatePluginApiKey(c echo.Context) error {
	pluginID := c.Param("id")
	keyID := c.Param("keyId")
	if pluginID == "" || keyID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "plugin id and key id are required"})
	}

	// Get address from JWT context
	address, ok := c.Get("address").(string)
	if !ok || address == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "authentication required"})
	}

	// Verify the requester is an admin of this plugin
	owner, err := s.queries.GetPluginOwnerWithRole(c.Request().Context(), &queries.GetPluginOwnerWithRoleParams{
		PluginID:  pluginID,
		PublicKey: address,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return c.JSON(http.StatusForbidden, map[string]string{"error": "not authorized to manage API keys for this plugin"})
		}
		s.logger.WithError(err).Error("failed to check plugin ownership")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}

	// Only admins can manage API keys
	if owner.Role != queries.PluginOwnerRoleAdmin {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "only admins can manage API keys"})
	}

	var req UpdateApiKeyRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	// Validate status
	if req.Status != 0 && req.Status != 1 {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "status must be 0 or 1"})
	}

	// Parse UUID
	keyUUID, err := uuid.Parse(keyID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid key id format"})
	}

	// Verify the key belongs to this plugin
	existingKey, err := s.queries.GetPluginApiKeyByID(c.Request().Context(), pgtype.UUID{Bytes: keyUUID, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "API key not found"})
		}
		s.logger.WithError(err).Error("failed to get API key")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}
	if string(existingKey.PluginID) != pluginID {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "API key does not belong to this plugin"})
	}

	ctx := c.Request().Context()
	var updated *queries.PluginApikey

	enabling := req.Status == 1 && existingKey.Status == 0
	if enabling {
		tx, err := s.pool.Begin(ctx)
		if err != nil {
			s.logger.WithError(err).Error("failed to begin transaction")
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to update API key"})
		}
		defer tx.Rollback(ctx)

		q := queries.New(tx)

		err = q.AcquireApiKeyLock(ctx, pluginID)
		if err != nil {
			s.logger.WithError(err).Error("failed to acquire lock")
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to update API key"})
		}

		count, err := q.CountActiveApiKeys(ctx, pluginID)
		if err != nil {
			s.logger.WithError(err).Error("failed to count API keys")
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to update API key"})
		}

		if count >= int64(s.cfg.MaxApiKeysPerPlugin) {
			return c.JSON(http.StatusConflict, map[string]string{"error": fmt.Sprintf("maximum number of API keys (%d) reached for this plugin", s.cfg.MaxApiKeysPerPlugin)})
		}

		updated, err = q.UpdatePluginApiKeyStatus(ctx, &queries.UpdatePluginApiKeyStatusParams{
			ID:     pgtype.UUID{Bytes: keyUUID, Valid: true},
			Status: req.Status,
		})
		if err != nil {
			s.logger.WithError(err).Error("failed to update API key status")
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to update API key"})
		}

		err = tx.Commit(ctx)
		if err != nil {
			s.logger.WithError(err).Error("failed to commit transaction")
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to update API key"})
		}
	} else {
		updated, err = s.queries.UpdatePluginApiKeyStatus(ctx, &queries.UpdatePluginApiKeyStatusParams{
			ID:     pgtype.UUID{Bytes: keyUUID, Valid: true},
			Status: req.Status,
		})
		if err != nil {
			s.logger.WithError(err).Error("failed to update API key status")
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to update API key"})
		}
	}

	var expiresAt *string
	if updated.ExpiresAt.Valid {
		t := updated.ExpiresAt.Time.Format(time.RFC3339)
		expiresAt = &t
	}

	s.logger.WithFields(logrus.Fields{
		"plugin_id": pluginID,
		"key_id":    keyID,
		"status":    req.Status,
	}).Info("API key status updated")

	return c.JSON(http.StatusOK, PluginApiKeyResponse{
		ID:        updated.ID.String(),
		PluginID:  string(updated.PluginID),
		ApiKey:    maskApiKey(updated.Apikey),
		CreatedAt: updated.CreatedAt.Time.Format(time.RFC3339),
		ExpiresAt: expiresAt,
		Status:    updated.Status,
	})
}

// DeletePluginApiKey expires an API key immediately (soft delete)
func (s *Server) DeletePluginApiKey(c echo.Context) error {
	pluginID := c.Param("id")
	keyID := c.Param("keyId")
	if pluginID == "" || keyID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "plugin id and key id are required"})
	}

	// Get address from JWT context
	address, ok := c.Get("address").(string)
	if !ok || address == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "authentication required"})
	}

	// Verify the requester owns this plugin and is an admin
	owner, err := s.queries.GetPluginOwnerWithRole(c.Request().Context(), &queries.GetPluginOwnerWithRoleParams{
		PluginID:  pluginID,
		PublicKey: address,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return c.JSON(http.StatusForbidden, map[string]string{"error": "not authorized to manage API keys for this plugin"})
		}
		s.logger.WithError(err).Error("failed to check plugin ownership")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}

	// Only admins can manage API keys
	if owner.Role != queries.PluginOwnerRoleAdmin {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "only admins can manage API keys"})
	}

	// Parse UUID
	keyUUID, err := uuid.Parse(keyID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid key id format"})
	}

	// Verify the key belongs to this plugin
	existingKey, err := s.queries.GetPluginApiKeyByID(c.Request().Context(), pgtype.UUID{Bytes: keyUUID, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "API key not found"})
		}
		s.logger.WithError(err).Error("failed to get API key")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}
	if string(existingKey.PluginID) != pluginID {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "API key does not belong to this plugin"})
	}

	// Expire the key (set expires_at to NOW())
	expired, err := s.queries.ExpirePluginApiKey(c.Request().Context(), pgtype.UUID{Bytes: keyUUID, Valid: true})
	if err != nil {
		s.logger.WithError(err).Error("failed to expire API key")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to delete API key"})
	}

	var expiresAt *string
	if expired.ExpiresAt.Valid {
		t := expired.ExpiresAt.Time.Format(time.RFC3339)
		expiresAt = &t
	}

	s.logger.WithFields(logrus.Fields{
		"plugin_id": pluginID,
		"key_id":    keyID,
	}).Info("API key expired (deleted)")

	return c.JSON(http.StatusOK, PluginApiKeyResponse{
		ID:        expired.ID.String(),
		PluginID:  string(expired.PluginID),
		ApiKey:    maskApiKey(expired.Apikey),
		CreatedAt: expired.CreatedAt.Time.Format(time.RFC3339),
		ExpiresAt: expiresAt,
		Status:    expired.Status,
	})
}

// EarningTransactionResponse is the API response for earning transactions
type EarningTransactionResponse struct {
	ID          string          `json:"id"`
	PluginID    string          `json:"pluginId"`
	PluginName  string          `json:"pluginName"`
	Amount      string          `json:"amount"`
	FeeAsset    itypes.FeeAsset `json:"fee_asset"`
	Type        string          `json:"type"`
	CreatedAt   string          `json:"createdAt"`
	FromAddress string          `json:"fromAddress"`
	TxHash      string          `json:"txHash"`
	Status      string          `json:"status"`
}

// EarningsResponse is the paginated API response for earnings
type EarningsResponse struct {
	Data       []EarningTransactionResponse `json:"data"`
	Page       int                          `json:"page"`
	Limit      int                          `json:"limit"`
	Total      int64                        `json:"total"`
	TotalPages int64                        `json:"totalPages"`
}

func (s *Server) GetEarnings(c echo.Context) error {
	address, ok := c.Get("address").(string)
	if !ok || address == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "authentication required"})
	}

	pluginID := c.QueryParam("pluginId")
	dateFrom := c.QueryParam("dateFrom")
	dateTo := c.QueryParam("dateTo")

	// TODO: Remove legacy response format once FE is updated to use pagination params
	pageParam := c.QueryParam("page")
	limitParam := c.QueryParam("limit")
	usePagination := pageParam != "" || limitParam != ""

	page := 1
	if pageParam != "" {
		if parsed, err := strconv.Atoi(pageParam); err == nil && parsed > 0 {
			page = parsed
		}
	}

	limit := 20
	if limitParam != "" {
		if parsed, err := strconv.Atoi(limitParam); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	offset := (page - 1) * limit

	var dateFromTs, dateToTs pgtype.Timestamptz
	if dateFrom != "" {
		t, err := time.Parse(time.RFC3339, dateFrom)
		if err == nil {
			dateFromTs = pgtype.Timestamptz{Time: t, Valid: true}
		}
	}
	if dateTo != "" {
		t, err := time.Parse(time.RFC3339, dateTo)
		if err == nil {
			dateToTs = pgtype.Timestamptz{Time: t, Valid: true}
		}
	}

	ctx := c.Request().Context()

	// TODO: Remove once FE uses pagination params
	queryLimit := int32(limit)
	queryOffset := int32(offset)
	if !usePagination {
		queryLimit = 10000
		queryOffset = 0
	}

	params := &queries.GetEarningsByPluginOwnerFilteredParams{
		PublicKey: address,
		Column2:   pluginID,
		Column3:   dateFromTs,
		Column4:   dateToTs,
		Limit:     queryLimit,
		Offset:    queryOffset,
	}

	earnings, err := s.queries.GetEarningsByPluginOwnerFiltered(ctx, params)
	if err != nil {
		s.logger.WithError(err).Error("failed to get earnings")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}

	data := make([]EarningTransactionResponse, len(earnings))
	for i, e := range earnings {
		pricingType := "per-tx"
		if pt, ok := e.PricingType.(string); ok {
			pricingType = pt
		}
		pid := ""
		if e.PluginID.Valid {
			pid = e.PluginID.String
		}
		data[i] = EarningTransactionResponse{
			ID:          strconv.FormatInt(e.ID, 10),
			PluginID:    pid,
			PluginName:  e.PluginName,
			Amount:      strconv.FormatInt(e.Amount, 10),
			FeeAsset:    itypes.DefaultFeeAsset,
			Type:        pricingType,
			CreatedAt:   e.CreatedAt.Time.Format(time.RFC3339),
			FromAddress: e.FromAddress,
			TxHash:      e.TxHash,
			Status:      e.Status,
		}
	}

	// TODO: Remove legacy response format once FE is updated to use pagination params
	if !usePagination {
		return c.JSON(http.StatusOK, data)
	}

	countParams := &queries.CountEarningsByPluginOwnerFilteredParams{
		PublicKey: address,
		Column2:   pluginID,
		Column3:   dateFromTs,
		Column4:   dateToTs,
	}
	total, err := s.queries.CountEarningsByPluginOwnerFiltered(ctx, countParams)
	if err != nil {
		s.logger.WithError(err).Error("failed to count earnings")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}

	totalPages := (total + int64(limit) - 1) / int64(limit)

	return c.JSON(http.StatusOK, EarningsResponse{
		Data:       data,
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	})
}

// PluginEarning represents earnings for a single plugin
type PluginEarning struct {
	Amount   string          `json:"amount"`
	FeeAsset itypes.FeeAsset `json:"fee_asset"`
}

// EarningsSummaryResponse is the API response for earnings summary
type EarningsSummaryResponse struct {
	TotalEarnings     PluginEarning            `json:"totalEarnings"`
	TotalTransactions int64                    `json:"totalTransactions"`
	EarningsByPlugin  map[string]PluginEarning `json:"earningsByPlugin"`
}

func (s *Server) GetEarningsSummary(c echo.Context) error {
	// Get address from JWT context (set by JWTAuthMiddleware)
	address, ok := c.Get("address").(string)
	if !ok || address == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "authentication required"})
	}

	// Get total summary (using Ethereum address from JWT)
	summary, err := s.queries.GetEarningsSummaryByPluginOwner(c.Request().Context(), address)
	if err != nil {
		s.logger.WithError(err).Error("failed to get earnings summary")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}

	// Get earnings by plugin
	byPlugin, err := s.queries.GetEarningsByPluginForOwner(c.Request().Context(), address)
	if err != nil {
		s.logger.WithError(err).Error("failed to get earnings by plugin")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}

	earningsByPlugin := make(map[string]PluginEarning)
	for _, p := range byPlugin {
		if p.PluginID.Valid {
			earningsByPlugin[p.PluginID.String] = PluginEarning{
				Amount:   strconv.FormatInt(p.Total, 10),
				FeeAsset: itypes.DefaultFeeAsset,
			}
		}
	}

	return c.JSON(http.StatusOK, EarningsSummaryResponse{
		TotalEarnings: PluginEarning{
			Amount:   strconv.FormatInt(summary.TotalEarnings, 10),
			FeeAsset: itypes.DefaultFeeAsset,
		},
		TotalTransactions: summary.TotalTransactions,
		EarningsByPlugin:  earningsByPlugin,
	})
}

// UpdatePluginRequest represents the request body for updating a plugin
type UpdatePluginRequest struct {
	Title          string `json:"title"`
	Description    string `json:"description"`
	ServerEndpoint string `json:"server_endpoint"`
	PayoutAddress  string `json:"payout_address"`

	// EIP-712 signature data
	Signature     string                    `json:"signature"`
	SignedMessage UpdatePluginSignedMessage `json:"signed_message"`
}

// UpdatePluginSignedMessage represents the EIP-712 message that was signed
type UpdatePluginSignedMessage struct {
	PluginID  string                `json:"pluginId"`
	Signer    string                `json:"signer"`
	Nonce     int64                 `json:"nonce"`
	Timestamp int64                 `json:"timestamp"`
	Updates   []sigutil.FieldUpdate `json:"updates"`
}

func (s *Server) UpdatePlugin(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "plugin id is required"})
	}

	var req UpdatePluginRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	// Validate signature is provided
	if req.Signature == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "signature is required"})
	}

	// Validate signed message matches the plugin ID
	if req.SignedMessage.PluginID != id {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "signed message plugin ID does not match"})
	}

	// Validate timestamp is recent (within 5 minutes)
	signedTime := time.Unix(req.SignedMessage.Timestamp, 0)
	if time.Since(signedTime) > 5*time.Minute {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "signature has expired"})
	}

	// Validate signer address format
	if !common.IsHexAddress(req.SignedMessage.Signer) {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid signer address"})
	}
	signerAddr := common.HexToAddress(req.SignedMessage.Signer)

	// Decode the signature
	sigHex := strings.TrimPrefix(req.Signature, "0x")
	sigBytes, err := hex.DecodeString(sigHex)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid signature format"})
	}

	// Reconstruct the typed data and verify the signature
	typedData := sigutil.PluginUpdateTypedData(
		req.SignedMessage.PluginID,
		req.SignedMessage.Signer,
		req.SignedMessage.Nonce,
		req.SignedMessage.Timestamp,
		req.SignedMessage.Updates,
	)

	valid, err := sigutil.VerifyEIP712Signature(signerAddr, typedData, sigBytes)
	if err != nil {
		s.logger.WithError(err).Error("failed to verify signature")
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "signature verification failed"})
	}
	if !valid {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "invalid signature"})
	}

	// Authorization check - verify signer owns this plugin and get their role
	owner, err := s.queries.GetPluginOwnerWithRole(c.Request().Context(), &queries.GetPluginOwnerWithRoleParams{
		PluginID:  id,
		PublicKey: signerAddr.Hex(),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return c.JSON(http.StatusForbidden, map[string]string{"error": "signer is not authorized to update this plugin"})
		}
		s.logger.WithError(err).Error("failed to check plugin ownership")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}

	// Role-based edit restrictions
	// Viewers cannot edit anything
	if owner.Role == queries.PluginOwnerRoleViewer {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "viewers cannot edit plugins"})
	}

	// Fetch existing plugin to validate unchanged fields match DB values
	existingPlugin, err := s.queries.GetPluginByID(c.Request().Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "plugin not found"})
		}
		s.logger.WithError(err).Error("failed to fetch existing plugin")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}

	// Build map of signed updates
	updateMap := make(map[string]sigutil.FieldUpdate)
	for _, u := range req.SignedMessage.Updates {
		updateMap[u.Field] = u
	}

	// Validate each field:
	// - If field is in updateMap (being changed):
	//   1. signed NewValue must match request value
	//   2. signed OldValue must match current DB value (prevents replay/stale-signature attacks)
	// - If field is NOT in updateMap (unchanged): request value must match existing DB value
	if fieldUpdate, ok := updateMap["title"]; ok {
		if fieldUpdate.NewValue != req.Title {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "title does not match signed value"})
		}
		if fieldUpdate.OldValue != existingPlugin.Title {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "title old value does not match current value"})
		}
	} else if req.Title != existingPlugin.Title {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "title change must be signed"})
	}

	if fieldUpdate, ok := updateMap["description"]; ok {
		if fieldUpdate.NewValue != req.Description {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "description does not match signed value"})
		}
		if fieldUpdate.OldValue != existingPlugin.Description {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "description old value does not match current value"})
		}
	} else if req.Description != existingPlugin.Description {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "description change must be signed"})
	}

	if fieldUpdate, ok := updateMap["serverEndpoint"]; ok {
		// Editors cannot modify server_endpoint
		if owner.Role == queries.PluginOwnerRoleEditor {
			return c.JSON(http.StatusForbidden, map[string]string{"error": "editors cannot modify server endpoint"})
		}
		if fieldUpdate.NewValue != req.ServerEndpoint {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "server_endpoint does not match signed value"})
		}
		if fieldUpdate.OldValue != existingPlugin.ServerEndpoint {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "server_endpoint old value does not match current value"})
		}
	} else if req.ServerEndpoint != existingPlugin.ServerEndpoint {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "server_endpoint change must be signed"})
	}

	// Validate payout address field
	if fieldUpdate, ok := updateMap["payoutAddress"]; ok {
		// Only admins can modify payout address
		if owner.Role != queries.PluginOwnerRoleAdmin {
			return c.JSON(http.StatusForbidden, map[string]string{"error": "only admins can modify payout address"})
		}

		// Validate and normalize new address
		normalizedNew := ""
		if req.PayoutAddress != "" {
			if !common.IsHexAddress(req.PayoutAddress) {
				return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid payout address format"})
			}
			normalizedNew = common.HexToAddress(req.PayoutAddress).Hex()
		}

		// Normalize old address for comparison
		normalizedOld := ""
		if existingPlugin.PayoutAddress.Valid && existingPlugin.PayoutAddress.String != "" {
			normalizedOld = common.HexToAddress(existingPlugin.PayoutAddress.String).Hex()
		}

		// Verify signed values match
		if fieldUpdate.NewValue != normalizedNew {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "payout address does not match signed value"})
		}
		if fieldUpdate.OldValue != normalizedOld {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "payout address old value does not match current value"})
		}
	} else if req.PayoutAddress != "" {
		// If not being updated, unsigned changes not allowed
		if !common.IsHexAddress(req.PayoutAddress) {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid payout address format"})
		}
		normalizedReq := common.HexToAddress(req.PayoutAddress).Hex()

		normalizedExisting := ""
		if existingPlugin.PayoutAddress.Valid && existingPlugin.PayoutAddress.String != "" {
			normalizedExisting = common.HexToAddress(existingPlugin.PayoutAddress.String).Hex()
		}

		if normalizedReq != normalizedExisting {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "payout address change must be signed"})
		}
	}

	// Normalize payout address before storing
	var normalizedPayoutAddress pgtype.Text
	if req.PayoutAddress != "" {
		normalizedPayoutAddress = pgtype.Text{
			String: common.HexToAddress(req.PayoutAddress).Hex(),
			Valid:  true,
		}
	}

	// Update the plugin with validated request values
	plugin, err := s.queries.UpdatePlugin(c.Request().Context(), &queries.UpdatePluginParams{
		ID:             id,
		Title:          req.Title,
		Description:    req.Description,
		ServerEndpoint: req.ServerEndpoint,
		PayoutAddress:  normalizedPayoutAddress,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "plugin not found"})
		}
		s.logger.WithError(err).Error("failed to update plugin")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}

	logFields := logrus.Fields{
		"plugin_id": id,
		"signer":    req.SignedMessage.Signer,
	}

	if _, ok := updateMap["payoutAddress"]; ok {
		logFields["payout_address_changed"] = true
		if plugin.PayoutAddress.Valid && plugin.PayoutAddress.String != "" {
			logFields["new_payout_address"] = maskPayoutAddress(plugin.PayoutAddress.String)
		}
	}

	s.logger.WithFields(logFields).Info("plugin updated successfully")

	return c.JSON(http.StatusOK, plugin)
}

// TeamMemberResponse represents a team member in API responses
type TeamMemberResponse struct {
	PublicKey     string `json:"publicKey"`
	Role          string `json:"role"`
	AddedVia      string `json:"addedVia"`
	AddedBy       string `json:"addedBy,omitempty"`
	CreatedAt     string `json:"createdAt"`
	IsCurrentUser bool   `json:"isCurrentUser"`
}

// ListTeamMembers returns team members for a plugin (admin only, excludes staff)
func (s *Server) ListTeamMembers(c echo.Context) error {
	pluginID := c.Param("id")
	if pluginID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "plugin id is required"})
	}

	// Get address from JWT context
	address, ok := c.Get("address").(string)
	if !ok || address == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "authentication required"})
	}

	// Check if the requester is an admin of this plugin
	owner, err := s.queries.GetPluginOwnerWithRole(c.Request().Context(), &queries.GetPluginOwnerWithRoleParams{
		PluginID:  pluginID,
		PublicKey: address,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return c.JSON(http.StatusForbidden, map[string]string{"error": "not authorized to view team members"})
		}
		s.logger.WithError(err).Error("failed to check plugin ownership")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}

	// Only admins can view team members
	if owner.Role != queries.PluginOwnerRoleAdmin {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "only admins can view team members"})
	}

	// Get team members (excludes staff)
	members, err := s.queries.ListPluginTeamMembers(c.Request().Context(), pluginID)
	if err != nil {
		s.logger.WithError(err).Error("failed to list team members")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}

	// Convert to response format
	response := make([]TeamMemberResponse, len(members))
	for i, m := range members {
		addedBy := ""
		if m.AddedByPublicKey.Valid {
			addedBy = m.AddedByPublicKey.String
		}
		response[i] = TeamMemberResponse{
			PublicKey:     m.PublicKey,
			Role:          string(m.Role),
			AddedVia:      string(m.AddedVia),
			AddedBy:       addedBy,
			CreatedAt:     m.CreatedAt.Time.Format(time.RFC3339),
			IsCurrentUser: m.PublicKey == address,
		}
	}

	return c.JSON(http.StatusOK, response)
}

// CreateInviteRequest represents a request to create an invite link
type CreateInviteRequest struct {
	Role string `json:"role"` // "editor" or "viewer"
}

// CreateInviteResponse represents the response with the invite link
type CreateInviteResponse struct {
	Link      string `json:"link"`
	ExpiresAt string `json:"expiresAt"`
	Role      string `json:"role"`
}

// CreateInvite generates a magic link to invite a new team member
func (s *Server) CreateInvite(c echo.Context) error {
	pluginID := c.Param("id")
	if pluginID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "plugin id is required"})
	}

	// Get address from JWT context
	address, ok := c.Get("address").(string)
	if !ok || address == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "authentication required"})
	}

	// Check if the requester is an admin of this plugin
	owner, err := s.queries.GetPluginOwnerWithRole(c.Request().Context(), &queries.GetPluginOwnerWithRoleParams{
		PluginID:  pluginID,
		PublicKey: address,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return c.JSON(http.StatusForbidden, map[string]string{"error": "not authorized to create invites"})
		}
		s.logger.WithError(err).Error("failed to check plugin ownership")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}

	// Only admins can create invites
	if owner.Role != queries.PluginOwnerRoleAdmin {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "only admins can create invites"})
	}

	var req CreateInviteRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	// Validate role
	if req.Role != "editor" && req.Role != "viewer" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "role must be 'editor' or 'viewer'"})
	}

	// Generate invite link
	link, _, err := s.inviteService.GenerateInviteLink(pluginID, req.Role, address)
	if err != nil {
		s.logger.WithError(err).Error("failed to generate invite link")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to generate invite link"})
	}

	expiresAt := time.Now().Add(InviteLinkExpiry).Format(time.RFC3339)

	s.logger.WithFields(logrus.Fields{
		"plugin_id":  pluginID,
		"role":       req.Role,
		"invited_by": address,
	}).Info("invite link created")

	return c.JSON(http.StatusCreated, CreateInviteResponse{
		Link:      link,
		ExpiresAt: expiresAt,
		Role:      req.Role,
	})
}

// ValidateInviteResponse represents the response when validating an invite
type ValidateInviteResponse struct {
	PluginID   string `json:"pluginId"`
	PluginName string `json:"pluginName"`
	Role       string `json:"role"`
	InvitedBy  string `json:"invitedBy"`
	ExpiresAt  string `json:"expiresAt"`
}

// ValidateInvite validates a magic link without accepting it (for preview)
func (s *Server) ValidateInvite(c echo.Context) error {
	data := c.QueryParam("data")
	sig := c.QueryParam("sig")

	if data == "" || sig == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "missing data or sig parameter"})
	}

	// Validate the invite link
	payload, err := s.inviteService.ValidateInviteLink(data, sig)
	if err != nil {
		if errors.Is(err, ErrInvalidSignature) {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid invite link"})
		}
		if errors.Is(err, ErrInviteExpired) {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invite link has expired"})
		}
		s.logger.WithError(err).Error("failed to validate invite")
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid invite link"})
	}

	// Check if link_id has already been used
	linkUUID, err := uuid.Parse(payload.LinkID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid invite link"})
	}
	used, err := s.queries.CheckLinkIdUsed(c.Request().Context(), pgtype.UUID{
		Bytes: linkUUID,
		Valid: true,
	})
	if err != nil {
		s.logger.WithError(err).Error("failed to check link usage")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}
	if used {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invite link has already been used"})
	}

	// Get plugin info
	plugin, err := s.queries.GetPluginByID(c.Request().Context(), payload.PluginID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "plugin not found"})
		}
		s.logger.WithError(err).Error("failed to get plugin")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}

	return c.JSON(http.StatusOK, ValidateInviteResponse{
		PluginID:   payload.PluginID,
		PluginName: plugin.Title,
		Role:       payload.Role,
		InvitedBy:  payload.InvitedBy,
		ExpiresAt:  time.Unix(payload.ExpiresAt, 0).Format(time.RFC3339),
	})
}

// AcceptInviteRequest represents the request to accept an invite
type AcceptInviteRequest struct {
	Data      string `json:"data"`      // Base64-encoded invite payload
	Signature string `json:"signature"` // Base64-encoded HMAC signature
}

// AcceptInvite accepts a magic link invite and adds the user as a team member
func (s *Server) AcceptInvite(c echo.Context) error {
	pluginID := c.Param("id")
	if pluginID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "plugin id is required"})
	}

	// Get address from JWT context (user must be authenticated)
	address, ok := c.Get("address").(string)
	if !ok || address == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "authentication required"})
	}

	var req AcceptInviteRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	if req.Data == "" || req.Signature == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "missing data or signature"})
	}

	// Validate the invite link
	payload, err := s.inviteService.ValidateInviteLink(req.Data, req.Signature)
	if err != nil {
		if errors.Is(err, ErrInvalidSignature) {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid invite link"})
		}
		if errors.Is(err, ErrInviteExpired) {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "invite link has expired"})
		}
		s.logger.WithError(err).Error("failed to validate invite")
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid invite link"})
	}

	// Verify plugin ID matches
	if payload.PluginID != pluginID {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invite is for a different plugin"})
	}

	// Check if link_id has already been used
	linkUUID := uuid.MustParse(payload.LinkID)
	used, err := s.queries.CheckLinkIdUsed(c.Request().Context(), pgtype.UUID{
		Bytes: linkUUID,
		Valid: true,
	})
	if err != nil {
		s.logger.WithError(err).Error("failed to check link usage")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}
	if used {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invite link has already been used"})
	}

	// Check if user is already a team member
	existingOwner, err := s.queries.GetPluginOwnerWithRole(c.Request().Context(), &queries.GetPluginOwnerWithRoleParams{
		PluginID:  pluginID,
		PublicKey: address,
	})
	if err == nil && existingOwner.Active {
		return c.JSON(http.StatusConflict, map[string]string{"error": "you are already a team member of this plugin"})
	}
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		s.logger.WithError(err).Error("failed to check existing membership")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}

	// Map role string to enum
	var role queries.PluginOwnerRole
	switch payload.Role {
	case "editor":
		role = queries.PluginOwnerRoleEditor
	case "viewer":
		role = queries.PluginOwnerRoleViewer
	default:
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid role in invite"})
	}

	// Add the team member
	_, err = s.queries.AddPluginTeamMember(c.Request().Context(), &queries.AddPluginTeamMemberParams{
		PluginID:         pluginID,
		PublicKey:        address,
		Role:             role,
		AddedByPublicKey: pgtype.Text{String: payload.InvitedBy, Valid: true},
		LinkID:           pgtype.UUID{Bytes: linkUUID, Valid: true},
	})
	if err != nil {
		s.logger.WithError(err).Error("failed to add team member")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to accept invite"})
	}

	s.logger.WithFields(logrus.Fields{
		"plugin_id":  pluginID,
		"public_key": address,
		"role":       payload.Role,
		"invited_by": payload.InvitedBy,
	}).Info("invite accepted, team member added")

	return c.JSON(http.StatusOK, map[string]string{
		"message": "invite accepted successfully",
		"role":    payload.Role,
	})
}

// RemoveTeamMember removes a team member from a plugin (admin only)
func (s *Server) RemoveTeamMember(c echo.Context) error {
	pluginID := c.Param("id")
	targetPublicKey := c.Param("publicKey")
	if pluginID == "" || targetPublicKey == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "plugin id and public key are required"})
	}

	// Get address from JWT context
	address, ok := c.Get("address").(string)
	if !ok || address == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "authentication required"})
	}

	// Check if the requester is an admin of this plugin
	owner, err := s.queries.GetPluginOwnerWithRole(c.Request().Context(), &queries.GetPluginOwnerWithRoleParams{
		PluginID:  pluginID,
		PublicKey: address,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return c.JSON(http.StatusForbidden, map[string]string{"error": "not authorized to remove team members"})
		}
		s.logger.WithError(err).Error("failed to check plugin ownership")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}

	// Only admins can remove team members
	if owner.Role != queries.PluginOwnerRoleAdmin {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "only admins can remove team members"})
	}

	// Cannot remove yourself
	if targetPublicKey == address {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "cannot remove yourself from the team"})
	}

	// Check target exists and is not an admin (admins cannot be removed via API)
	target, err := s.queries.GetPluginOwnerWithRole(c.Request().Context(), &queries.GetPluginOwnerWithRoleParams{
		PluginID:  pluginID,
		PublicKey: targetPublicKey,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "team member not found"})
		}
		s.logger.WithError(err).Error("failed to get target member")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}

	// Cannot remove other admins or staff
	if target.Role == queries.PluginOwnerRoleAdmin || target.Role == queries.PluginOwnerRoleStaff {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "cannot remove admin or staff members"})
	}

	// Remove the team member
	err = s.queries.RemovePluginTeamMember(c.Request().Context(), &queries.RemovePluginTeamMemberParams{
		PluginID:  pluginID,
		PublicKey: targetPublicKey,
	})
	if err != nil {
		s.logger.WithError(err).Error("failed to remove team member")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to remove team member"})
	}

	s.logger.WithFields(logrus.Fields{
		"plugin_id":  pluginID,
		"removed_by": address,
		"removed":    targetPublicKey,
	}).Info("team member removed")

	return c.JSON(http.StatusOK, map[string]string{"message": "team member removed successfully"})
}

// KillSwitchResponse represents the kill switch status for a plugin
type KillSwitchResponse struct {
	PluginID       string `json:"pluginId"`
	KeygenEnabled  bool   `json:"keygenEnabled"`
	KeysignEnabled bool   `json:"keysignEnabled"`
}

// GetKillSwitch returns the kill switch status for a plugin (staff only)
func (s *Server) GetKillSwitch(c echo.Context) error {
	pluginID := c.Param("id")
	if pluginID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "plugin id is required"})
	}

	// Get address from JWT context
	address, ok := c.Get("address").(string)
	if !ok || address == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "authentication required"})
	}

	// Check if the requester is a staff member of this plugin
	owner, err := s.queries.GetPluginOwnerWithRole(c.Request().Context(), &queries.GetPluginOwnerWithRoleParams{
		PluginID:  pluginID,
		PublicKey: address,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return c.JSON(http.StatusForbidden, map[string]string{"error": "not authorized to view kill switch"})
		}
		s.logger.WithError(err).Error("failed to check plugin ownership")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}

	// Only staff or admin can view kill switch
	if owner.Role != queries.PluginOwnerRoleStaff && owner.Role != queries.PluginOwnerRoleAdmin {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "only staff can view kill switch"})
	}

	// Get control flags for this plugin
	keygenKey := pluginID + "-keygen"
	keysignKey := pluginID + "-keysign"

	flags, err := s.queries.GetControlFlagsByKeys(c.Request().Context(), []string{keygenKey, keysignKey})
	if err != nil {
		s.logger.WithError(err).Error("failed to get control flags")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to get kill switch status"})
	}

	// Build a map for easy lookup
	flagMap := make(map[string]bool, len(flags))
	for _, f := range flags {
		flagMap[f.Key] = f.Enabled
	}

	// Default to enabled if no flag exists
	keygenEnabled := true
	if enabled, ok := flagMap[keygenKey]; ok {
		keygenEnabled = enabled
	}

	keysignEnabled := true
	if enabled, ok := flagMap[keysignKey]; ok {
		keysignEnabled = enabled
	}

	return c.JSON(http.StatusOK, KillSwitchResponse{
		PluginID:       pluginID,
		KeygenEnabled:  keygenEnabled,
		KeysignEnabled: keysignEnabled,
	})
}

// SetKillSwitchRequest represents a request to set kill switch status
type SetKillSwitchRequest struct {
	KeygenEnabled  *bool `json:"keygenEnabled"`
	KeysignEnabled *bool `json:"keysignEnabled"`
}

// SetKillSwitch sets the kill switch status for a plugin (staff only)
func (s *Server) SetKillSwitch(c echo.Context) error {
	pluginID := c.Param("id")
	if pluginID == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "plugin id is required"})
	}

	// Get address from JWT context
	address, ok := c.Get("address").(string)
	if !ok || address == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "authentication required"})
	}

	// Check if the requester is a staff member of this plugin
	owner, err := s.queries.GetPluginOwnerWithRole(c.Request().Context(), &queries.GetPluginOwnerWithRoleParams{
		PluginID:  pluginID,
		PublicKey: address,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return c.JSON(http.StatusForbidden, map[string]string{"error": "not authorized to set kill switch"})
		}
		s.logger.WithError(err).Error("failed to check plugin ownership")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}

	// Only staff or admin can set kill switch
	if owner.Role != queries.PluginOwnerRoleStaff && owner.Role != queries.PluginOwnerRoleAdmin {
		return c.JSON(http.StatusForbidden, map[string]string{"error": "only staff can set kill switch"})
	}

	// Parse request body
	var req SetKillSwitchRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "invalid request body"})
	}

	// At least one flag must be provided
	if req.KeygenEnabled == nil && req.KeysignEnabled == nil {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "at least one of keygenEnabled or keysignEnabled must be provided"})
	}

	// Update control flags
	keygenKey := pluginID + "-keygen"
	keysignKey := pluginID + "-keysign"

	if req.KeygenEnabled != nil {
		if err := s.queries.UpsertControlFlag(c.Request().Context(), &queries.UpsertControlFlagParams{
			Key:     keygenKey,
			Enabled: *req.KeygenEnabled,
		}); err != nil {
			s.logger.WithError(err).Error("failed to update keygen flag")
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to update kill switch"})
		}
	}

	if req.KeysignEnabled != nil {
		if err := s.queries.UpsertControlFlag(c.Request().Context(), &queries.UpsertControlFlagParams{
			Key:     keysignKey,
			Enabled: *req.KeysignEnabled,
		}); err != nil {
			s.logger.WithError(err).Error("failed to update keysign flag")
			return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to update kill switch"})
		}
	}

	// Get updated state
	flags, err := s.queries.GetControlFlagsByKeys(c.Request().Context(), []string{keygenKey, keysignKey})
	if err != nil {
		s.logger.WithError(err).Error("failed to get control flags after update")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to confirm kill switch status"})
	}

	// Build a map for easy lookup
	flagMap := make(map[string]bool, len(flags))
	for _, f := range flags {
		flagMap[f.Key] = f.Enabled
	}

	keygenEnabled := true
	if enabled, ok := flagMap[keygenKey]; ok {
		keygenEnabled = enabled
	}

	keysignEnabled := true
	if enabled, ok := flagMap[keysignKey]; ok {
		keysignEnabled = enabled
	}

	s.logger.WithFields(logrus.Fields{
		"plugin_id":       pluginID,
		"staff":           address,
		"keygen_enabled":  keygenEnabled,
		"keysign_enabled": keysignEnabled,
	}).Info("kill switch updated")

	return c.JSON(http.StatusOK, KillSwitchResponse{
		PluginID:       pluginID,
		KeygenEnabled:  keygenEnabled,
		KeysignEnabled: keysignEnabled,
	})
}
