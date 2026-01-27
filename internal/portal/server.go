package portal

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/sirupsen/logrus"

	"github.com/vultisig/verifier/config"
	"github.com/vultisig/verifier/internal/sigutil"
	"github.com/vultisig/verifier/internal/storage/postgres/queries"
	"github.com/vultisig/vultisig-go/address"
	vcommon "github.com/vultisig/vultisig-go/common"
)

type Server struct {
	cfg         config.PortalConfig
	queries     *queries.Queries
	logger      *logrus.Logger
	authService *PortalAuthService
}

func NewServer(cfg config.PortalConfig, pool *pgxpool.Pool) *Server {
	logger := logrus.WithField("service", "portal").Logger
	return &Server{
		cfg:         cfg,
		queries:     queries.New(pool),
		logger:      logger,
		authService: NewPortalAuthService(cfg.Server.JWTSecret, logger),
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

	// Plugin routes (public - no auth required)
	e.GET("/plugins", s.ListPlugins)
	e.GET("/plugins/:id", s.GetPlugin)
	e.PUT("/plugins/:id", s.UpdatePlugin)
	e.GET("/plugins/:id/pricings", s.GetPluginPricings)

	// Protected routes (require JWT auth)
	protected := e.Group("")
	protected.Use(s.JWTAuthMiddleware)
	protected.GET("/plugins/:id/api-keys", s.GetPluginApiKeys)
	protected.GET("/earnings", s.GetEarnings)
	protected.GET("/earnings/summary", s.GetEarningsSummary)
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

	plugin, err := s.queries.GetPluginByID(c.Request().Context(), queries.PluginID(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "plugin not found"})
		}
		s.logger.WithError(err).Error("failed to get plugin")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}

	return c.JSON(http.StatusOK, plugin)
}

func (s *Server) ListPlugins(c echo.Context) error {
	plugins, err := s.queries.ListPlugins(c.Request().Context())
	if err != nil {
		s.logger.WithError(err).Error("failed to list plugins")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}

	return c.JSON(http.StatusOK, plugins)
}

// PluginPricingResponse is the API response for plugin pricing
type PluginPricingResponse struct {
	ID        string  `json:"id"`
	PluginID  string  `json:"pluginId"`
	Asset     string  `json:"asset"`
	Type      string  `json:"type"`
	Frequency *string `json:"frequency"`
	Amount    int64   `json:"amount"`
	Metric    string  `json:"metric"`
}

func (s *Server) GetPluginPricings(c echo.Context) error {
	id := c.Param("id")
	if id == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "plugin id is required"})
	}

	pricings, err := s.queries.GetPluginPricings(c.Request().Context(), queries.PluginID(id))
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
			Asset:     string(p.Asset),
			Type:      string(p.Type),
			Frequency: freq,
			Amount:    p.Amount,
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

	// Verify the requester owns this plugin (using Ethereum address from JWT)
	_, err := s.queries.GetPluginOwner(c.Request().Context(), &queries.GetPluginOwnerParams{
		PluginID:  queries.PluginID(id),
		PublicKey: address,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return c.JSON(http.StatusForbidden, map[string]string{"error": "not authorized to view API keys for this plugin"})
		}
		s.logger.WithError(err).Error("failed to check plugin ownership")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}

	apiKeys, err := s.queries.GetPluginApiKeys(c.Request().Context(), queries.PluginID(id))
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
			ApiKey:    k.Apikey,
			CreatedAt: k.CreatedAt.Time.Format(time.RFC3339),
			ExpiresAt: expiresAt,
			Status:    k.Status,
		}
	}

	return c.JSON(http.StatusOK, response)
}

// EarningTransactionResponse is the API response for earning transactions
type EarningTransactionResponse struct {
	ID          string `json:"id"`
	PluginID    string `json:"pluginId"`
	PluginName  string `json:"pluginName"`
	Amount      int64  `json:"amount"`
	Asset       string `json:"asset"`
	Type        string `json:"type"`
	CreatedAt   string `json:"createdAt"`
	FromAddress string `json:"fromAddress"`
	TxHash      string `json:"txHash"`
	Status      string `json:"status"`
}

func (s *Server) GetEarnings(c echo.Context) error {
	// Get address from JWT context (set by JWTAuthMiddleware)
	address, ok := c.Get("address").(string)
	if !ok || address == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "authentication required"})
	}

	// Parse filter parameters
	pluginID := c.QueryParam("pluginId")
	dateFrom := c.QueryParam("dateFrom")
	dateTo := c.QueryParam("dateTo")

	// Build filter params (using Ethereum address from JWT)
	params := &queries.GetEarningsByPluginOwnerFilteredParams{
		PublicKey: address,
		Column2:   pluginID,
	}

	if dateFrom != "" {
		t, err := time.Parse(time.RFC3339, dateFrom)
		if err == nil {
			params.Column3 = pgtype.Timestamptz{Time: t, Valid: true}
		}
	}

	if dateTo != "" {
		t, err := time.Parse(time.RFC3339, dateTo)
		if err == nil {
			params.Column4 = pgtype.Timestamptz{Time: t, Valid: true}
		}
	}

	earnings, err := s.queries.GetEarningsByPluginOwnerFiltered(c.Request().Context(), params)
	if err != nil {
		s.logger.WithError(err).Error("failed to get earnings")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}

	// Convert to API response format
	response := make([]EarningTransactionResponse, len(earnings))
	for i, e := range earnings {
		pricingType := "per-tx"
		if pt, ok := e.PricingType.(string); ok {
			pricingType = pt
		}
		pluginID := ""
		if e.PluginID.Valid {
			pluginID = e.PluginID.String
		}
		response[i] = EarningTransactionResponse{
			ID:          strconv.FormatInt(e.ID, 10),
			PluginID:    pluginID,
			PluginName:  e.PluginName,
			Amount:      e.Amount,
			Asset:       e.Asset,
			Type:        pricingType,
			CreatedAt:   e.CreatedAt.Time.Format(time.RFC3339),
			FromAddress: e.FromAddress,
			TxHash:      e.TxHash,
			Status:      e.Status,
		}
	}

	return c.JSON(http.StatusOK, response)
}

// EarningsSummaryResponse is the API response for earnings summary
type EarningsSummaryResponse struct {
	TotalEarnings     int64            `json:"totalEarnings"`
	TotalTransactions int64            `json:"totalTransactions"`
	EarningsByPlugin  map[string]int64 `json:"earningsByPlugin"`
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

	earningsByPlugin := make(map[string]int64)
	for _, p := range byPlugin {
		if p.PluginID.Valid {
			earningsByPlugin[p.PluginID.String] = p.Total
		}
	}

	return c.JSON(http.StatusOK, EarningsSummaryResponse{
		TotalEarnings:     summary.TotalEarnings,
		TotalTransactions: summary.TotalTransactions,
		EarningsByPlugin:  earningsByPlugin,
	})
}

// UpdatePluginRequest represents the request body for updating a plugin
type UpdatePluginRequest struct {
	Title          string `json:"title"`
	Description    string `json:"description"`
	ServerEndpoint string `json:"server_endpoint"`

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

	// Authorization check - verify signer owns this plugin
	_, err = s.queries.GetPluginOwner(c.Request().Context(), &queries.GetPluginOwnerParams{
		PluginID:  queries.PluginID(id),
		PublicKey: signerAddr.Hex(),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return c.JSON(http.StatusForbidden, map[string]string{"error": "signer is not authorized to update this plugin"})
		}
		s.logger.WithError(err).Error("failed to check plugin ownership")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}

	// Fetch existing plugin to validate unchanged fields match DB values
	existingPlugin, err := s.queries.GetPluginByID(c.Request().Context(), queries.PluginID(id))
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
		if fieldUpdate.NewValue != req.ServerEndpoint {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "server_endpoint does not match signed value"})
		}
		if fieldUpdate.OldValue != existingPlugin.ServerEndpoint {
			return c.JSON(http.StatusBadRequest, map[string]string{"error": "server_endpoint old value does not match current value"})
		}
	} else if req.ServerEndpoint != existingPlugin.ServerEndpoint {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "server_endpoint change must be signed"})
	}

	// Update the plugin with validated request values
	plugin, err := s.queries.UpdatePlugin(c.Request().Context(), &queries.UpdatePluginParams{
		ID:             queries.PluginID(id),
		Title:          req.Title,
		Description:    req.Description,
		ServerEndpoint: req.ServerEndpoint,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return c.JSON(http.StatusNotFound, map[string]string{"error": "plugin not found"})
		}
		s.logger.WithError(err).Error("failed to update plugin")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}

	s.logger.WithFields(logrus.Fields{
		"plugin_id": id,
		"signer":    req.SignedMessage.Signer,
	}).Info("plugin updated successfully")

	return c.JSON(http.StatusOK, plugin)
}
