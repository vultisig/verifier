package api

import (
	"context"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"
	"github.com/vultisig/verifier/types"
)

func (s *Server) GetPublicKeyFees(c echo.Context) error {
	pluginID, ok := c.Get("plugin_id").(types.PluginID)
	if !ok || pluginID == "" {
		return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage(msgRequiredPluginID))
	}
	if pluginID != types.PluginVultisigFees_feee {
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
	}

	return c.JSON(http.StatusOK, status)
}
