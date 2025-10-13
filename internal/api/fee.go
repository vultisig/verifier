package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/vultisig/verifier/types"
)

func (s *Server) GetFees(c echo.Context) error {
	pluginId := fmt.Sprint(c.Get("plugin_id"))
	since := c.QueryParam("since")
	var sinceTime *time.Time
	if since != "" {
		st, err := time.Parse(time.RFC3339, since)
		if err != nil {
			return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage("invalid since time"))
		}
		sinceTime = &st
	}

	if pluginId != string(types.PluginVultisigFees_feee) {
		return c.JSON(http.StatusUnauthorized, NewErrorResponseWithMessage("unauthorized"))
	}

	history, err := s.feeService.GetFeeInfo(c.Request().Context(), sinceTime)
	if err != nil {
		s.logger.WithError(err).Errorf("Failed to get fees: %s ", err)
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage("failed to get fees"))
	}

	status := http.StatusOK
	return c.JSON(status, NewSuccessResponse(status, history))
}

func (s *Server) MarkCollected(c echo.Context) error {
	var req struct {
		IDs         []uuid.UUID `json:"ids"`
		TxHash      string      `json:"tx_hash"`
		CollectedAt time.Time   `json:"collected_at"`
	}
	if err := c.Bind(&req); err != nil {
		s.logger.WithError(err).Error("Failed to parse request body for MarkCollected")
		return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage("failed to parse request"))
	}

	fees, err := s.feeService.MarkFeesCollected(c.Request().Context(), req.CollectedAt, req.IDs, req.TxHash)
	if err != nil {
		s.logger.WithError(err).Error("Failed to mark fees as collected")
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage("failed to mark fees as collected"))
	}

	return c.JSON(http.StatusOK, NewSuccessResponse(http.StatusOK, fees))
}
