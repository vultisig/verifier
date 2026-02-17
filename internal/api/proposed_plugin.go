package api

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

func (s *Server) ValidateProposedPlugin(c echo.Context) error {
	pluginID := c.Param("pluginId")
	if pluginID == "" {
		return s.badRequest(c, msgRequiredPluginID, nil)
	}

	approved, err := s.db.IsProposedPluginApproved(c.Request().Context(), pluginID)
	if err != nil {
		return s.internal(c, msgProposedPluginValidateFailed, err)
	}
	if !approved {
		return c.JSON(http.StatusNotFound, NewErrorResponseWithMessage(msgProposedPluginNotApproved))
	}

	return c.JSON(http.StatusOK, NewSuccessResponse(http.StatusOK, map[string]bool{"valid": true}))
}
