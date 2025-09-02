package api

import (
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/vultisig/verifier/types"
)

func (s *Server) GetPublicKeyFees(c echo.Context) error {
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

	publicKey := c.Param("publicKey")

	history, err := s.feeService.PublicKeyGetFeeInfo(c.Request().Context(), publicKey, sinceTime)
	if err != nil {
		s.logger.WithError(err).Errorf("Failed to get fees for public key: %s", publicKey)
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage("failed to get fees"))
	}

	status := http.StatusOK
	return c.JSON(status, NewSuccessResponse(status, history))
}

func (s *Server) GetFeeBalance(c echo.Context) error {
	publicKey := c.Param("publicKey")

	balance, err := s.feeService.GetFeeBalance(c.Request().Context(), publicKey)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage("failed to get fee balance"))
	}

	return c.JSON(http.StatusOK, NewSuccessResponse(http.StatusOK, map[string]interface{}{
		"balance":    balance,
		"public_key": publicKey,
	}))
}

func (s *Server) CreateFeeCollectionBatch(c echo.Context) error {
	s.feeService.SignRequestMutex.Lock()
	defer s.feeService.SignRequestMutex.Unlock()

	type request struct {
		PublicKey string `json:"public_key"`
	}
	var req request

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage("invalid request"))
	}

	feesOwed, err := s.feeService.CreateFeeCollectionBatch(c.Request().Context(), req.PublicKey)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage("failed to get fee balance"))
	}

	return c.JSON(http.StatusOK, NewSuccessResponse(http.StatusOK, feesOwed))
}

func (s *Server) UpdateFeeCollectionBatch(c echo.Context) error {
	s.feeService.SignRequestMutex.Lock()
	defer s.feeService.SignRequestMutex.Unlock()

	type request struct {
		BatchID   uuid.UUID            `json:"batch_id"`
		TxHash    string               `json:"tx_hash"`
		Status    types.FeeBatchStatus `json:"status"`
		PublicKey string               `json:"public_key"`
	}
	var req request

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage("invalid request"))
	}

	err := s.feeService.UpdateFeeCollectionBatch(c.Request().Context(), req.PublicKey, req.BatchID, req.TxHash, req.Status)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage("failed to update fee collection batch"))
	}

	return c.JSON(http.StatusOK, NewSuccessResponse(http.StatusOK, map[string]string{
		"batch_id":   req.BatchID.String(),
		"tx_hash":    req.TxHash,
		"status":     string(req.Status),
		"public_key": req.PublicKey,
	}))

}

func (s *Server) GetDraftFeeBatches(c echo.Context) error {
	publicKey := c.Param("publicKey")

	batches, err := s.db.GetFeeBatchesByStateAndPublicKey(c.Request().Context(), publicKey, types.FeeBatchStatusDraft)
	if err != nil {
		s.logger.WithError(err).Errorf("Failed to get draft fee batches for public key: %s", publicKey)
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage("failed to get draft fee batches"))
	}

	return c.JSON(http.StatusOK, NewSuccessResponse(http.StatusOK, batches))
}
