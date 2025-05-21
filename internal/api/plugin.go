package api

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"github.com/vultisig/verifier/internal/types"
	ptypes "github.com/vultisig/verifier/types"
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

	plugins, err := s.db.FindPlugins(c.Request().Context(), take, skip, sort)
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

	plugin, err := s.db.FindPluginById(c.Request().Context(), ptypes.PluginID(pluginID))
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
		return c.JSON(http.StatusBadRequest, NewErrorResponse("invalid request"))
	}

	if err := c.Validate(&plugin); err != nil {
		return c.JSON(http.StatusBadRequest, NewErrorResponse(err.Error()))
	}

	updated, err := s.db.UpdatePlugin(c.Request().Context(), ptypes.PluginID(pluginID), plugin)
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

	if err := s.db.DeletePluginById(c.Request().Context(), ptypes.PluginID(pluginID)); err != nil {
		s.logger.WithError(err).Error("Failed to delete plugin")
		return c.JSON(http.StatusInternalServerError, NewErrorResponse("failed to delete plugin"))
	}

	return c.NoContent(http.StatusNoContent)
}

func (s *Server) GetCategories(c echo.Context) error {
	categories, err := s.db.FindCategories(c.Request().Context())
	if err != nil {
		message := echo.Map{
			"message": "failed to get categories",
		}
		s.logger.Error(err)
		return c.JSON(http.StatusInternalServerError, message)
	}

	return c.JSON(http.StatusOK, categories)
}
