package api

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"
	"github.com/microcosm-cc/bluemonday"
	"github.com/vultisig/verifier/common"

	"github.com/vultisig/verifier/internal/types"
	ptypes "github.com/vultisig/verifier/types"
)

func (s *Server) GetAllPluginPolicies(c echo.Context) error {
	publicKey := c.Request().Header.Get("public_key")
	if publicKey == "" {
		err := fmt.Errorf("missing required header: public_key")
		message := map[string]interface{}{
			"message": "failed to get policies",
			"error":   err.Error(),
		}
		s.logger.Error(err)
		return c.JSON(http.StatusBadRequest, message)
	}

	pluginType := c.Request().Header.Get("plugin_type")
	if pluginType == "" {
		err := fmt.Errorf("missing required header: plugin_type")
		message := map[string]interface{}{
			"message": "failed to get policies",
			"error":   err.Error(),
		}
		s.logger.Error(err)
		return c.JSON(http.StatusBadRequest, message)
	}

	skip, err := strconv.Atoi(c.QueryParam("skip"))

	if err != nil {
		skip = 0
	}

	take, err := strconv.Atoi(c.QueryParam("take"))

	if err != nil {
		take = 20
	}

	if take > 100 {
		take = 100
	}

	policies, err := s.policyService.GetPluginPolicies(c.Request().Context(), pluginType, publicKey, take, skip)
	if err != nil {
		message := map[string]interface{}{
			"message": fmt.Sprintf("failed to get policies for public_key: %s", publicKey),
		}
		s.logger.Error(err)
		return c.JSON(http.StatusInternalServerError, message)
	}

	return c.JSON(http.StatusOK, policies)
}

func (s *Server) GetPlugins(c echo.Context) error {
	skip, err := strconv.Atoi(c.QueryParam("skip"))

	if err != nil {
		skip = 0
	}

	take, err := strconv.Atoi(c.QueryParam("take"))

	if err != nil {
		take = 20
	}

	if take > 100 {
		take = 100
	}

	sort := c.QueryParam("sort")

	filters := types.PluginFilters{
		Term:       common.GetQueryParam(c, "term"),
		TagID:      common.GetUUIDParam(c, "tag_id"),
		CategoryID: common.GetUUIDParam(c, "category_id"),
	}

	plugins, err := s.db.FindPlugins(c.Request().Context(), filters, take, skip, sort)

	if err != nil {
		message := echo.Map{
			"message": "failed to get plugins",
		}
		s.logger.Error(err)
		return c.JSON(http.StatusInternalServerError, message)
	}

	return c.JSON(http.StatusOK, plugins)
}

func (s *Server) GetPlugin(c echo.Context) error {
	pluginID := c.Param("pluginId")
	if pluginID == "" {
		err := fmt.Errorf("plugin id is required")
		message := echo.Map{
			"message": "failed to get plugin",
			"error":   err.Error(),
		}
		s.logger.Error(err)

		return c.JSON(http.StatusBadRequest, message)
	}

	plugin, err := s.pluginService.GetPluginWithRating(c.Request().Context(), pluginID)
	if err != nil {
		message := echo.Map{
			"message": "failed to get plugin",
		}
		s.logger.Error(err)
		return c.JSON(http.StatusInternalServerError, message)
	}

	return c.JSON(http.StatusOK, plugin)
}

func (s *Server) CreatePlugin(c echo.Context) error {
	var plugin types.PluginCreateDto
	if err := c.Bind(&plugin); err != nil {
		return fmt.Errorf("fail to parse request, err: %w", err)
	}

	if err := c.Validate(&plugin); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"message": err.Error(),
		})
	}

	created, err := s.pluginService.CreatePluginWithRating(c.Request().Context(), plugin)
	if err != nil {
		message := echo.Map{
			"message": "failed to create plugin",
		}
		s.logger.Error(err)
		return c.JSON(http.StatusInternalServerError, message)
	}

	return c.JSON(http.StatusOK, created)
}

func (s *Server) UpdatePlugin(c echo.Context) error {
	pluginID := c.Param("pluginId")
	if pluginID == "" {
		message := echo.Map{
			"message": "failed to update plugin",
			"error":   "plugin id is required",
		}
		return c.JSON(http.StatusBadRequest, message)
	}

	var plugin types.PluginUpdateDto
	if err := c.Bind(&plugin); err != nil {
		return fmt.Errorf("fail to parse request, err: %w", err)
	}

	if err := c.Validate(&plugin); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"message": err.Error(),
		})
	}

	updated, err := s.db.UpdatePlugin(c.Request().Context(), ptypes.PluginID(pluginID), plugin)
	if err != nil {
		message := echo.Map{
			"message": "failed to update plugin",
		}
		s.logger.Error(err)
		return c.JSON(http.StatusInternalServerError, message)
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

func (s *Server) GetTags(c echo.Context) error {
	tags, err := s.db.FindTags(c.Request().Context())
	if err != nil {
		message := echo.Map{
			"message": "failed to get tags",
		}
		s.logger.Error(err)
		return c.JSON(http.StatusInternalServerError, message)
	}
	return c.JSON(http.StatusOK, tags)
}

func (s *Server) AttachPluginTag(c echo.Context) error {
	pluginID := c.Param("pluginId")
	if pluginID == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"message": "failed to find plugin",
			"error":   "plugin id is required",
		})
	}
	_, err := s.db.FindPluginById(c.Request().Context(), nil, ptypes.PluginID(pluginID))
	if err != nil {
		s.logger.Error(err)
		return c.JSON(http.StatusBadRequest, echo.Map{
			"message": "failed to find plugin",
			"error":   "plugin not found",
		})
	}

	var createTagDto types.CreateTagDto
	if err := c.Bind(&createTagDto); err != nil {
		return fmt.Errorf("fail to parse request, err: %w", err)
	}
	if err := c.Validate(&createTagDto); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"message": err.Error(),
		})
	}

	var tag *types.Tag
	tag, err = s.db.FindTagByName(c.Request().Context(), createTagDto.Name)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			tag, err = s.db.CreateTag(c.Request().Context(), createTagDto)
			if err != nil {
				s.logger.Error(err)
				return c.JSON(http.StatusInternalServerError, echo.Map{
					"message": "failed to create tag",
				})
			}
		} else {
			s.logger.Error(err)
			return c.JSON(http.StatusInternalServerError, echo.Map{
				"message": "failed to check for existing tag",
			})
		}
	}

	updatedPlugin, err := s.db.AttachTagToPlugin(c.Request().Context(), ptypes.PluginID(pluginID), tag.ID)
	if err != nil {
		s.logger.Error(err)
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"message": "failed to attach tag",
		})
	}

	return c.JSON(http.StatusOK, updatedPlugin)
}

func (s *Server) DetachPluginTag(c echo.Context) error {
	pluginID := c.Param("pluginId")
	if pluginID == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"message": "failed to find plugin",
			"error":   "plugin id is required",
		})
	}
	_, err := s.db.FindPluginById(c.Request().Context(), nil, ptypes.PluginID(pluginID))
	if err != nil {
		s.logger.Error(err)
		return c.JSON(http.StatusBadRequest, echo.Map{
			"message": "failed to find plugin",
			"error":   "plugin not found",
		})
	}

	tagID := c.Param("tagId")
	if tagID == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"message": "failed to find tag",
			"error":   "tag id is required",
		})
	}
	tag, err := s.db.FindTagById(c.Request().Context(), tagID)
	if err != nil {
		s.logger.Error(err)
		return c.JSON(http.StatusBadRequest, echo.Map{
			"message": "failed to find tag",
			"error":   "tag not found",
		})
	}

	updatedPlugin, err := s.db.DetachTagFromPlugin(c.Request().Context(), ptypes.PluginID(pluginID), tag.ID)
	if err != nil {
		s.logger.Error(err)
		return c.JSON(http.StatusInternalServerError, echo.Map{
			"message": "failed to detach tag",
		})
	}

	return c.JSON(http.StatusOK, updatedPlugin)
}

func (s *Server) GetPluginPolicyTransactionHistory(c echo.Context) error {
	policyID := c.Param("policyId")

	if policyID == "" {
		err := fmt.Errorf("policy ID is required")
		message := map[string]interface{}{
			"message": "failed to get policy",
			"error":   err.Error(),
		}
		return c.JSON(http.StatusBadRequest, message)
	}

	skip, err := strconv.Atoi(c.QueryParam("skip"))

	if err != nil {
		skip = 0
	}

	take, err := strconv.Atoi(c.QueryParam("take"))

	if err != nil {
		take = 20
	}

	if take > 100 {
		take = 100
	}

	policyHistory, err := s.policyService.GetPluginPolicyTransactionHistory(c.Request().Context(), policyID, take, skip)
	if err != nil {
		err = fmt.Errorf("failed to get policy history: %w", err)
		message := map[string]interface{}{
			"message": fmt.Sprintf("failed to get policy history: %s", policyID),
		}
		s.logger.Error(err)
		return c.JSON(http.StatusInternalServerError, message)
	}

	return c.JSON(http.StatusOK, policyHistory)
}

func (s *Server) CreateReview(c echo.Context) error {
	var review types.ReviewCreateDto
	if err := c.Bind(&review); err != nil {
		return fmt.Errorf("fail to parse request, err: %w", err)
	}

	if err := c.Validate(&review); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{
			"message": err.Error(),
		})
	}

	// If allowing HTML, sanitize with bluemonday:
	p := bluemonday.UGCPolicy()
	review.Comment = p.Sanitize(review.Comment)

	pluginID := c.Param("pluginId")
	if pluginID == "" {
		err := fmt.Errorf("plugin id is required")
		message := echo.Map{
			"message": "failed to get plugin",
			"error":   err.Error(),
		}
		s.logger.Error(err)

		return c.JSON(http.StatusBadRequest, message)
	}

	created, err := s.pluginService.CreatePluginReviewWithRating(c.Request().Context(), review, pluginID)
	if err != nil {
		message := echo.Map{
			"message": "failed to create review",
		}
		s.logger.Error(err)
		return c.JSON(http.StatusInternalServerError, message)
	}

	return c.JSON(http.StatusOK, created)
}

func (s *Server) GetReviews(c echo.Context) error {
	pluginId := c.Param("pluginId")
	if pluginId == "" {
		err := fmt.Errorf("plugin id is required")
		message := echo.Map{
			"message": "failed to get plugin",
			"error":   err.Error(),
		}
		s.logger.Error(err)

		return c.JSON(http.StatusBadRequest, message)
	}

	skip, err := strconv.Atoi(c.QueryParam("skip"))

	if err != nil {
		skip = 0
	}

	take, err := strconv.Atoi(c.QueryParam("take"))

	if err != nil {
		take = 20
	}

	if take > 100 {
		take = 100
	}

	sort := c.QueryParam("sort")

	reviews, err := s.db.FindReviews(c.Request().Context(), pluginId, skip, take, sort)
	if err != nil {
		message := echo.Map{
			"message": "failed to get reviews",
		}
		s.logger.Error(err)
		return c.JSON(http.StatusInternalServerError, message)
	}

	return c.JSON(http.StatusOK, reviews)
}
