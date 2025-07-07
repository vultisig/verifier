package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func (s *Server) GetPublicKeyFees(c echo.Context) error {
	publicKey := c.Param("publicKey")

	history, err := s.feeService.PublicKeyGetFeeInfo(c.Request().Context(), publicKey)
	if err != nil {
		s.logger.WithError(err).Errorf("Failed to get fees for public key: %s", publicKey)
		return c.JSON(http.StatusInternalServerError, NewErrorResponse(http.StatusInternalServerError, "failed to get fees", err.Error()))
	}

	status := http.StatusOK
	return c.JSON(status, NewSuccessResponse(status, history))
}
