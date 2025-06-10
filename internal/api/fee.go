package api

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"
)

func (s *Server) GetPluginPolicyFees(c echo.Context) error {
	fmt.Println("HIT")
	policyID := c.Param("policyId")

	history, err := s.policyService.PluginPolicyGetFeeInfo(c.Request().Context(), policyID)
	if err != nil {
		s.logger.WithError(err).Errorf("Failed to get fees for policy ID: %s", policyID)
		return c.JSON(http.StatusInternalServerError, NewErrorResponse("failed to get fees"))
	}

	return c.JSON(http.StatusOK, history)
}
