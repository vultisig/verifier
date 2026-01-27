package portal

import (
	"encoding/hex"
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
)

type Server struct {
	cfg     config.PortalConfig
	queries *queries.Queries
	logger  *logrus.Logger
}

func NewServer(cfg config.PortalConfig, pool *pgxpool.Pool) *Server {
	return &Server{
		cfg:     cfg,
		queries: queries.New(pool),
		logger:  logrus.WithField("service", "portal").Logger,
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

	// Plugin routes
	e.GET("/plugins", s.ListPlugins)
	e.GET("/plugins/:id", s.GetPlugin)
	e.PUT("/plugins/:id", s.UpdatePlugin)
	e.GET("/plugins/:id/pricings", s.GetPluginPricings)
	e.GET("/plugins/:id/api-keys", s.GetPluginApiKeys)

	// Earnings routes
	e.GET("/earnings", s.GetEarnings)
	e.GET("/earnings/summary", s.GetEarningsSummary)
}

func (s *Server) Healthz(c echo.Context) error {
	return c.String(http.StatusOK, "OK")
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

	// Get public key from header for authorization
	publicKey := c.Request().Header.Get("X-Public-Key")
	if publicKey == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "public key required"})
	}

	// Verify the requester owns this plugin
	_, err := s.queries.GetPluginOwner(c.Request().Context(), &queries.GetPluginOwnerParams{
		PluginID:  queries.PluginID(id),
		PublicKey: publicKey,
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
	// Get public key from header for authorization
	publicKey := c.Request().Header.Get("X-Public-Key")
	if publicKey == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "public key required"})
	}

	// Parse filter parameters
	pluginID := c.QueryParam("pluginId")
	dateFrom := c.QueryParam("dateFrom")
	dateTo := c.QueryParam("dateTo")

	// Build filter params
	params := &queries.GetEarningsByPluginOwnerFilteredParams{
		PublicKey: publicKey,
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
		response[i] = EarningTransactionResponse{
			ID:          strconv.FormatInt(e.ID, 10),
			PluginID:    e.PluginID.String,
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
	// Get public key from header for authorization
	publicKey := c.Request().Header.Get("X-Public-Key")
	if publicKey == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{"error": "public key required"})
	}

	// Get total summary
	summary, err := s.queries.GetEarningsSummaryByPluginOwner(c.Request().Context(), publicKey)
	if err != nil {
		s.logger.WithError(err).Error("failed to get earnings summary")
		return c.JSON(http.StatusInternalServerError, map[string]string{"error": "internal server error"})
	}

	// Get earnings by plugin
	byPlugin, err := s.queries.GetEarningsByPluginForOwner(c.Request().Context(), publicKey)
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
	Signature     string                  `json:"signature"`
	SignedMessage UpdatePluginSignedMessage `json:"signed_message"`
}

// UpdatePluginSignedMessage represents the EIP-712 message that was signed
type UpdatePluginSignedMessage struct {
	PluginID  string               `json:"pluginId"`
	Signer    string               `json:"signer"`
	Nonce     int64                `json:"nonce"`
	Timestamp int64                `json:"timestamp"`
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

	// Validate that the updates in the signed message match what's being requested
	updateMap := make(map[string]string)
	for _, u := range req.SignedMessage.Updates {
		updateMap[u.Field] = u.NewValue
	}

	// Check that requested values match signed values
	if title, ok := updateMap["title"]; ok && title != req.Title {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "title does not match signed value"})
	}
	if desc, ok := updateMap["description"]; ok && desc != req.Description {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "description does not match signed value"})
	}
	if endpoint, ok := updateMap["serverEndpoint"]; ok && endpoint != req.ServerEndpoint {
		return c.JSON(http.StatusBadRequest, map[string]string{"error": "server_endpoint does not match signed value"})
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

	// Update the plugin
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
