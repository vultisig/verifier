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

func (s *Server) RevertFeeTransaction(c echo.Context) error {
	s.feeService.SignRequestMutex.Lock()
	defer s.feeService.SignRequestMutex.Unlock()

	pluginId := fmt.Sprint(c.Get("plugin_id"))
	batchIdString := c.Param("batch_id")
	if pluginId != string(types.PluginVultisigFees_feee) {
		return c.JSON(http.StatusUnauthorized, NewErrorResponseWithMessage("unauthorized"))
	}

	batchId, err := uuid.Parse(batchIdString)
	if err != nil {
		return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage("invalid batch id"))
	}

	fees, err := s.db.GetFeeCreditsByIds(c.Request().Context(), []uuid.UUID{batchId})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage("failed to get fees"))
	}
	if len(fees) != 1 {
		return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage("invalid batch id"))
	}

	revertAmount := fees[0].Amount

	debitId := uuid.New()
	err = s.feeService.CreateFeeDebit(c.Request().Context(), nil, types.FeeDebit{
		Fee: types.Fee{
			ID:        debitId,
			Amount:    revertAmount,
			PublicKey: fees[0].PublicKey,
			Ref:       fmt.Sprintf("batch_id:%s", batchId),
		},
		Type: types.FeeDebitTypeFailedTx,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage("failed to create fee debit"))
	}

	return c.JSON(http.StatusOK, NewSuccessResponse(http.StatusOK, map[string]string{
		"debit_id": debitId.String(),
	}))

}

func (s *Server) CreateFeeCredit(c echo.Context) error {
	s.feeService.SignRequestMutex.Lock()
	defer s.feeService.SignRequestMutex.Unlock()

	type request struct {
		Amount    int64     `json:"amount"`
		PublicKey string    `json:"public_key"`
		ID        uuid.UUID `json:"id"`
	}
	var req request

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage("invalid request"))
	}

	feesOwed, err := s.feeService.GetFeeBalanceUnlocked(c.Request().Context(), req.PublicKey)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage("failed to get fee balance"))
	}

	if feesOwed < req.Amount {
		return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage("insufficient balance"))
	}

	if err := s.feeService.CreateFeeCredit(c.Request().Context(), nil, types.FeeCredit{
		Fee: types.Fee{
			ID:        req.ID,
			Amount:    uint64(req.Amount),
			PublicKey: req.PublicKey,
		},
		Type: types.FeeCreditTypeFeeTransacted,
	}); err != nil {
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage("failed to create fee credit"))
	}

	return c.JSON(http.StatusOK, NewSuccessResponse(http.StatusOK, map[string]string{
		"id": req.ID.String(),
	}))

}
