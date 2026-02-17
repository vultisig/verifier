package api

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"

	"github.com/vultisig/verifier/internal/conv"
	itypes "github.com/vultisig/verifier/internal/types"
	"github.com/vultisig/verifier/types"
)

func (s *Server) GetPublicKeyFees(c echo.Context) error {
	pluginID, ok := c.Get("plugin_id").(string)
	if !ok || pluginID == "" {
		return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage(msgRequiredPluginID))
	}
	if pluginID != itypes.PluginVultisigFees {
		return c.JSON(http.StatusUnauthorized, NewErrorResponseWithMessage("unauthorized"))
	}

	publicKey := c.Param("publicKey")

	var (
		isTrialActive bool
		err           error
	)
	err = s.db.WithTransaction(c.Request().Context(), func(ctx context.Context, tx pgx.Tx) error {
		isTrialActive, _, err = s.db.IsTrialActive(ctx, tx, publicKey)
		return err
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgGetFeesFailed))
	}

	if isTrialActive {
		return c.JSON(http.StatusOK, NewSuccessResponse(http.StatusOK, []*types.Fee{}))
	}

	fees, err := s.feeService.PublicKeyGetFeeInfo(c.Request().Context(), publicKey)
	if err != nil {
		s.logger.WithError(err).Errorf("Failed to get fees for public key: %s", publicKey)
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgGetFeesFailed))
	}

	return c.JSON(http.StatusOK, NewSuccessResponse(http.StatusOK, fees))
}

func (s *Server) MarkCollected(c echo.Context) error {
	var req struct {
		IDs     []uint64 `json:"ids" validate:"required,min=1"`
		TxHash  string   `json:"tx_hash" validate:"required"`
		Network string   `json:"network" validate:"required"`
		Amount  uint64   `json:"amount" validate:"required,gt=0"`
	}
	if err := c.Bind(&req); err != nil {
		s.logger.WithError(err).Error("Failed to parse request body for MarkCollected")
		return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage(msgRequestParseFailed))
	}

	err := s.feeService.MarkFeesCollected(c.Request().Context(), req.IDs, req.Network, req.TxHash, req.Amount)
	if err != nil {
		s.logger.WithError(err).Error("Failed to mark fees as collected")
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgMarkFeesCollectedFailed))
	}

	return c.JSON(http.StatusOK, "OK")
}

func (s *Server) IssueCredit(c echo.Context) error {
	var req struct {
		PublicKey string `json:"public_key" validate:"required"`
		Amount    uint64 `json:"amount" validate:"required,gt=0"`
		Reason    string `json:"reason" validate:"required"`
	}

	if err := c.Bind(&req); err != nil {
		s.logger.WithError(err).Error("Failed to parse request body for IssueCredit")
		return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage(msgRequestParseFailed))
	}

	err := s.feeService.IssueCredit(c.Request().Context(), req.PublicKey, req.Amount, req.Reason)
	if err != nil {
		s.logger.WithError(err).Error("Failed to issue credit")
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgIssueCreditFailed))
	}

	return c.JSON(http.StatusOK, "OK")
}

func (s *Server) GetUserFees(c echo.Context) error {
	publicKey, ok := c.Get("vault_public_key").(string)
	if !ok || publicKey == "" {
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgVaultPublicKeyGetFailed))
	}

	status, err := s.feeService.GetUserFees(c.Request().Context(), publicKey)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get user fees")
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgGetUserFeesFailed))
	}

	err = s.db.WithTransaction(c.Request().Context(), func(ctx context.Context, tx pgx.Tx) error {
		status.IsTrialActive, status.TrialRemaining, err = s.db.IsTrialActive(ctx, tx, publicKey)
		return err
	})
	if err != nil {
		s.logger.WithError(err).Warnf("Failed to check trial info")
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgGetUserTrialInfo))
	}

	return c.JSON(http.StatusOK, status)
}

// GetPluginFeeHistory returns paginated fee history for a specific plugin
func (s *Server) GetPluginFeeHistory(c echo.Context) error {
	pluginID := c.Param("pluginId")
	if pluginID == "" {
		return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage(msgRequiredPluginID))
	}

	skip, take, err := conv.PageParamsFromCtx(c, 0, 20)
	if err != nil {
		return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage(msgInvalidPagination))
	}

	publicKey, ok := c.Get("vault_public_key").(string)
	if !ok || publicKey == "" {
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgVaultPublicKeyGetFailed))
	}

	fees, totalCount, err := s.db.GetFeesByPluginID(
		c.Request().Context(),
		pluginID,
		publicKey,
		skip,
		take,
	)
	if err != nil {
		s.logger.WithError(err).Errorf("s.db.GetFeesByPluginID: %s", pluginID)
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgGetFeesFailed))
	}

	// Build title map for the single plugin
	titleMap, err := s.pluginService.GetPluginTitlesByIDs(c.Request().Context(), []string{pluginID})
	if err != nil {
		s.logger.WithError(err).Error("s.pluginService.GetPluginTitlesByIDs")
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgGetPluginFailed))
	}

	return c.JSON(http.StatusOK, NewSuccessResponse(http.StatusOK, itypes.FeeHistoryPaginatedList{
		History:    itypes.FromFeesWithStatus(fees, titleMap),
		TotalCount: totalCount,
	}))
}

// GetPluginBillingSummary returns billing summary for all plugins the user has used
func (s *Server) GetPluginBillingSummary(c echo.Context) error {
	publicKey, ok := c.Get("vault_public_key").(string)
	if !ok || publicKey == "" {
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgVaultPublicKeyGetFailed))
	}

	rows, err := s.db.GetPluginBillingSummary(c.Request().Context(), publicKey)
	if err != nil {
		s.logger.WithError(err).Error("s.db.GetPluginBillingSummary")
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgGetFeesFailed))
	}

	if len(rows) == 0 {
		return c.JSON(http.StatusOK, NewSuccessResponse(http.StatusOK, itypes.PluginBillingSummaryList{
			Plugins:    []itypes.PluginBillingSummary{},
			TotalCount: 0,
		}))
	}

	// Collect plugin IDs for lookups
	pluginIDs := make([]string, len(rows))
	for i, row := range rows {
		pluginIDs[i] = row.PluginID
	}

	titleMap, err := s.pluginService.GetPluginTitlesByIDs(c.Request().Context(), pluginIDs)
	if err != nil {
		s.logger.WithError(err).Error("s.pluginService.GetPluginTitlesByIDs")
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgGetPluginFailed))
	}

	// Get pricings for all plugins
	pricingsMap, err := s.db.GetPricingsByPluginIDs(c.Request().Context(), pluginIDs)
	if err != nil {
		s.logger.WithError(err).Error("s.db.GetPricingsByPluginIDs")
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgGetFeesFailed))
	}

	// Convert rows to response format
	summaries := make([]itypes.PluginBillingSummary, len(rows))
	for i, row := range rows {
		pricings := pricingsMap[row.PluginID]
		summaries[i] = itypes.PluginBillingSummary{
			PluginID:    row.PluginID,
			AppName:     titleMap[row.PluginID],
			Pricing:     formatPricings(pricings),
			StartDate:   row.StartDate.UTC(),
			NextPayment: calculateNextPaymentFromPricings(pricings, row.StartDate),
			TotalFees:   strconv.FormatUint(row.TotalFees, 10),
		}
	}

	return c.JSON(http.StatusOK, NewSuccessResponse(http.StatusOK, itypes.PluginBillingSummaryList{
		Plugins:    summaries,
		TotalCount: uint32(len(summaries)),
	}))
}

// formatPricings formats multiple pricing entries into a combined string
// Example output: "0.50 USDC one-time + 0.01 USDC per transaction"
func formatPricings(pricings []itypes.PricingInfo) string {
	if len(pricings) == 0 {
		return ""
	}

	parts := make([]string, 0, len(pricings))
	for _, p := range pricings {
		parts = append(parts, formatSinglePricing(p))
	}

	return strings.Join(parts, " + ")
}

// formatSinglePricing formats a single pricing entry for display
func formatSinglePricing(p itypes.PricingInfo) string {
	// Convert from smallest unit (6 decimals for USDC)
	amountFloat := float64(p.Amount) / 1_000_000
	assetUpper := strings.ToUpper(p.Asset)

	switch p.Type {
	case "per-tx":
		return formatAmount(amountFloat) + " " + assetUpper + " per transaction"
	case "once":
		return formatAmount(amountFloat) + " " + assetUpper + " one-time"
	case "recurring":
		if p.Frequency != nil {
			return formatAmount(amountFloat) + " " + assetUpper + " / " + *p.Frequency
		}
		return formatAmount(amountFloat) + " " + assetUpper + " recurring"
	default:
		return formatAmount(amountFloat) + " " + assetUpper
	}
}

// formatAmount formats a float amount, removing unnecessary decimals
func formatAmount(amount float64) string {
	if amount == float64(int64(amount)) {
		return strconv.FormatInt(int64(amount), 10)
	}
	return fmt.Sprintf("%.2f", amount)
}

// calculateNextPaymentFromPricings finds the next payment date from recurring pricings
func calculateNextPaymentFromPricings(pricings []itypes.PricingInfo, startDate time.Time) *time.Time {
	var earliest *time.Time

	for _, p := range pricings {
		if p.Type != "recurring" || p.Frequency == nil {
			continue
		}

		next := calculateNextPaymentForFrequency(startDate, *p.Frequency)
		if next == nil {
			continue
		}

		if earliest == nil || next.Before(*earliest) {
			earliest = next
		}
	}

	return earliest
}

// calculateNextPaymentForFrequency calculates the next payment date for a given frequency
func calculateNextPaymentForFrequency(startDate time.Time, frequency string) *time.Time {
	now := time.Now().UTC()
	next := startDate.UTC()

	// Advance until we find the next payment date after now
	for next.Before(now) || next.Equal(now) {
		switch frequency {
		case "daily":
			next = next.AddDate(0, 0, 1)
		case "weekly":
			next = next.AddDate(0, 0, 7)
		case "biweekly":
			next = next.AddDate(0, 0, 14)
		case "monthly":
			next = next.AddDate(0, 1, 0)
		default:
			return nil
		}
	}

	return &next
}
