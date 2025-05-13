package api

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/vultisig/verifier/internal/types"
)

func (s *Server) GetPlugins(c echo.Context) error {
	skip, err := strconv.Atoi(c.QueryParam("skip"))

	if err != nil {
		skip = 0
	}

	take, err := strconv.Atoi(c.QueryParam("take"))

	if err != nil {
		take = 999
	}

	sort := c.QueryParam("sort")

	plugins, err := s.db.FindPlugins(c.Request().Context(), skip, take, sort)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get plugins")
		return c.JSON(http.StatusInternalServerError, NewErrorResponse("failed to get plugins"))
	}

	return c.JSON(http.StatusOK, plugins)
}

func (s *Server) GetPlugin(c echo.Context) error {
	pluginID := c.Param("pluginId")
	if pluginID == "" {
		return c.JSON(http.StatusBadRequest, NewErrorResponse("failed to get plugin"))
	}
	uPluginID, err := uuid.Parse(pluginID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, NewErrorResponse("failed to get plugin"))
	}
	plugin, err := s.db.FindPluginById(c.Request().Context(), uPluginID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get plugin")
		return c.JSON(http.StatusInternalServerError, NewErrorResponse("failed to get plugin"))
	}

	return c.JSON(http.StatusOK, plugin)
}

func (s *Server) CreatePlugin(c echo.Context) error {
	var plugin types.PluginCreateDto
	if err := c.Bind(&plugin); err != nil {
		return c.JSON(http.StatusBadRequest, NewErrorResponse("invalid request"))
	}

	if err := c.Validate(&plugin); err != nil {
		return c.JSON(http.StatusBadRequest, NewErrorResponse(err.Error()))
	}

	created, err := s.db.CreatePlugin(c.Request().Context(), plugin)
	if err != nil {
		s.logger.WithError(err).Error("Failed to create plugin")
		return c.JSON(http.StatusInternalServerError, NewErrorResponse("failed to create plugin"))
	}

	return c.JSON(http.StatusOK, created)
}

func (s *Server) UpdatePlugin(c echo.Context) error {
	pluginID := c.Param("pluginId")
	if pluginID == "" {
		return c.JSON(http.StatusBadRequest, NewErrorResponse("plugin id is required"))
	}

	var plugin types.PluginUpdateDto
	if err := c.Bind(&plugin); err != nil {
		return fmt.Errorf("fail to parse request, err: %w", err)
	}

	if err := c.Validate(&plugin); err != nil {
		return c.JSON(http.StatusBadRequest, NewErrorResponse(err.Error()))
	}
	uPluginID, err := uuid.Parse(pluginID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, NewErrorResponse("invalid plugin id"))
	}
	updated, err := s.db.UpdatePlugin(c.Request().Context(), uPluginID, plugin)
	if err != nil {
		s.logger.WithError(err).Error("Failed to update plugin")
		return c.JSON(http.StatusInternalServerError, NewErrorResponse("failed to update plugin"))
	}

	return c.JSON(http.StatusOK, updated)
}

func (s *Server) DeletePlugin(c echo.Context) error {
	pluginID := c.Param("pluginId")
	if pluginID == "" {
		return c.JSON(http.StatusBadRequest, NewErrorResponse("plugin id is required"))
	}
	uPluginID, err := uuid.Parse(pluginID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, NewErrorResponse("invalid plugin id"))
	}
	if err := s.db.DeletePluginById(c.Request().Context(), uPluginID); err != nil {
		s.logger.WithError(err).Error("Failed to delete plugin")
		return c.JSON(http.StatusInternalServerError, NewErrorResponse("failed to delete plugin"))
	}

	return c.NoContent(http.StatusNoContent)
}
