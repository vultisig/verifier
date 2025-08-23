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

	batchIdString := c.Param("batch_id")

	batchId, err := uuid.Parse(batchIdString)
	if err != nil {
		return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage("invalid batch id"))
	}

	creditFee, err := s.db.GetCreditTxByBatchId(c.Request().Context(), batchId)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage("failed to get fees"))
	}

	if creditFee == nil {
		return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage("invalid batch id"))
	}

	revertAmount := creditFee.Amount

	debitId := uuid.New()
	dbTx, err := s.db.Pool().Begin(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage("failed to begin transaction"))
	}

	_, err = s.db.InsertFeeDebitTx(c.Request().Context(), dbTx, types.FeeDebit{
		Fee: types.Fee{
			ID:        debitId,
			Amount:    revertAmount,
			PublicKey: creditFee.PublicKey,
			Ref:       fmt.Sprintf("batch_id:%s", batchId),
			Type:      types.FeeTypeDebit,
		},
		Subtype: types.FeeDebitSubtypeTypeFailedTx,
	})
	if err != nil {
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage("failed to create fee debit"))
	}

	if err != nil {
		dbTx.Rollback(c.Request().Context())
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage("failed to insert fee debit"))
	}

	if err := dbTx.Commit(c.Request().Context()); err != nil {
		dbTx.Rollback(c.Request().Context())
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage("failed to commit transaction"))
	}

	return c.JSON(http.StatusOK, NewSuccessResponse(http.StatusOK, map[string]string{
		"debit_id": debitId.String(),
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
