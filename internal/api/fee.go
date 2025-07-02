package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func (s *Server) GetPluginPolicyFees(c echo.Context) error {

	policyID := c.Param("policyId")

	history, err := s.feeService.PluginPolicyGetFeeInfo(c.Request().Context(), policyID)

	//TODO garry appropriate error code responses. For now everything is 500
	if err != nil {
		s.logger.WithError(err).Errorf("Failed to get fees for policy ID: %s", policyID)
		return c.JSON(http.StatusInternalServerError, NewErrorResponse(http.StatusInternalServerError, "failed to get fees", err.Error()))
	}

	status := http.StatusOK
	return c.JSON(status, NewSuccessResponse(status, history))
}

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

func (s *Server) GetPluginFees(c echo.Context) error {
	pluginID := c.Param("pluginId")

	history, err := s.feeService.PluginGetFeeInfo(c.Request().Context(), pluginID)
	if err != nil {
		s.logger.WithError(err).Errorf("Failed to get fees for plugin ID: %s", pluginID)
		return c.JSON(http.StatusInternalServerError, NewErrorResponse(http.StatusInternalServerError, "failed to get fees", err.Error()))
	}

	status := http.StatusOK
	return c.JSON(status, NewSuccessResponse(status, history))
}

func (s *Server) GetAllFees(c echo.Context) error {

	history, err := s.feeService.GetAllFeeInfo(c.Request().Context())
	if err != nil {
		return c.JSON(http.StatusInternalServerError, NewErrorResponse(http.StatusInternalServerError, "failed to get fees", err.Error()))
	}

	status := http.StatusOK
	return c.JSON(status, NewSuccessResponse(status, history))
}
