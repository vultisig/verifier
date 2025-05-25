package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"
	"github.com/microcosm-cc/bluemonday"
	"github.com/vultisig/verifier/common"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/vultisig/recipes/ethereum"
	rtypes "github.com/vultisig/recipes/types"
	"github.com/vultisig/recipes/util"
	"github.com/vultisig/verifier/internal/tasks"
	"github.com/vultisig/verifier/internal/types"
	ptypes "github.com/vultisig/verifier/types"
)

func (s *Server) SignPluginMessages(c echo.Context) error {
	s.logger.Debug("PLUGIN SERVER: SIGN MESSAGES")

	var req ptypes.PluginKeysignRequest
	if err := c.Bind(&req); err != nil {
		return fmt.Errorf("fail to parse request, err: %w", err)
	}

	// Plugin-specific validations
	if len(req.Messages) != 1 {
		return fmt.Errorf("plugin signing requires exactly one message hash, current: %d", len(req.Messages))
	}

	policyUUID, err := uuid.Parse(req.PolicyID)
	if err != nil {
		return fmt.Errorf("failed to parse policy ID: %w", err)
	}

	// Get policy from database
	policy, err := s.db.GetPluginPolicy(c.Request().Context(), policyUUID)
	if err != nil {
		return fmt.Errorf("failed to get policy from database: %w", err)
	}

	// Validate policy matches plugin
	if policy.PluginID != ptypes.PluginID(req.PluginID) {
		return fmt.Errorf("policy plugin ID mismatch")
	}

	var recipe rtypes.Policy
	if err := protojson.Unmarshal([]byte(policy.Recipe), &recipe); err != nil {
		return fmt.Errorf("failed to unmarshal recipe: %w", err)
	}

	// Presently we can only verify for payroll
	if policy.PluginID != ptypes.PluginVultisigPayroll_0000 {
		return fmt.Errorf("unsupported plugin ID: %s", policy.PluginID)
	}

	transactionAllowedByPolicy := false

	// Payroll only builds ethereum txs right now
	ethChain := ethereum.NewEthereum()
	decodedTx, err := ethChain.ParseTransaction(req.Transaction)
	if err != nil {
		return fmt.Errorf("failed to parse transaction: %w", err)
	}

	for _, rule := range recipe.GetRules() { // Use generated GetRules() getter
		if rule == nil { // Defensive check
			continue
		}
		resourcePathString := rule.GetResource() // Use generated getter
		resourcePath, err := util.ParseResource(resourcePathString)
		if err != nil {
			log.Printf("Skipping rule %s: invalid resource path %s: %v", rule.GetId(), resourcePathString, err)
			continue
		}

		if resourcePath.ChainId != "ethereum" {
			log.Printf("Skipping rule %s: target chain %s is not 'ethereum'", rule.GetId(), resourcePath.ChainId)
			continue
		}

		fmt.Printf("\nEvaluating rule: %s (Description: %s)\n", rule.GetId(), rule.GetDescription())
		fmt.Printf("  Targeting: Chain='%s', Asset='%s', Function='%s'\n",
			resourcePath.ChainId, resourcePath.ProtocolId, resourcePath.FunctionId)

		protocol, err := ethChain.GetProtocol(resourcePath.ProtocolId)
		if err != nil {
			log.Printf("  Skipping rule %s: Could not get protocol for asset '%s': %v", rule.GetId(), resourcePath.ProtocolId, err)
			continue
		}
		fmt.Printf("  Using protocol: %s (ID: %s)\n", protocol.Name(), protocol.ID())

		policyMatcher := &rtypes.PolicyFunctionMatcher{
			FunctionID:  resourcePath.FunctionId,
			Constraints: rule.GetParameterConstraints(), // Use generated getter
		}

		matches, _, err := protocol.MatchFunctionCall(decodedTx, policyMatcher)
		if err != nil {
			log.Printf("  Error during transaction matching for rule %s, function %s: %v", rule.GetId(), resourcePath.FunctionId, err)
			continue
		}

		if matches {
			fmt.Printf("  SUCCESS: Transaction matches rule %s for function %s!\n", rule.GetId(), resourcePath.FunctionId)
			transactionAllowedByPolicy = true
		}
	}

	if !transactionAllowedByPolicy {
		return fmt.Errorf("transaction does not match any rule in the policy")
	}

	// Reuse existing signing logic
	result, err := s.redis.Get(c.Request().Context(), req.SessionID)
	if err == nil && result != "" {
		return c.NoContent(http.StatusOK)
	}

	if err := s.redis.Set(c.Request().Context(), req.SessionID, req.SessionID, 30*time.Minute); err != nil {
		s.logger.Errorf("fail to set session, err: %v", err)
	}

	buf, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("fail to marshal to json, err: %w", err)
	}

	ti, err := s.client.EnqueueContext(c.Request().Context(),
		asynq.NewTask(tasks.TypeKeySignDKLS, buf),
		asynq.MaxRetry(0),
		asynq.Timeout(2*time.Minute),
		asynq.Retention(5*time.Minute),
		asynq.Queue(tasks.QUEUE_NAME))

	if err != nil {
		return fmt.Errorf("fail to enqueue keysign task: %w", err)
	}

	return c.JSON(http.StatusOK, ti.ID)
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
		s.logger.WithError(err).Error("Failed to get plugins")
		return c.JSON(http.StatusInternalServerError, NewErrorResponse("failed to get plugins"))
	}

	return c.JSON(http.StatusOK, plugins)
}

func (s *Server) GetPlugin(c echo.Context) error {
	pluginID := c.Param("pluginId")
	if pluginID == "" {
		s.logger.Error("plugin id is required")
		return c.JSON(http.StatusBadRequest, NewErrorResponse("plugin id is required"))
	}

	plugin, err := s.pluginService.GetPluginWithRating(c.Request().Context(), pluginID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get plugin")
		return c.JSON(http.StatusInternalServerError, NewErrorResponse("failed to get plugin"))
	}

	return c.JSON(http.StatusOK, plugin)
}

func (s *Server) CreatePlugin(c echo.Context) error {
	var plugin types.PluginCreateDto
	if err := c.Bind(&plugin); err != nil {
		s.logger.WithError(err).Error("Failed to parse request")
		return c.JSON(http.StatusBadRequest, NewErrorResponse("failed to parse request"))
	}

	if err := c.Validate(&plugin); err != nil {
		return c.JSON(http.StatusBadRequest, NewErrorResponse(err.Error()))
	}

	created, err := s.pluginService.CreatePluginWithRating(c.Request().Context(), plugin)
	if err != nil {
		s.logger.WithError(err).Error("Failed to create plugin")
		return c.JSON(http.StatusInternalServerError, NewErrorResponse("failed to create plugin"))
	}

	return c.JSON(http.StatusOK, created)
}

func (s *Server) UpdatePlugin(c echo.Context) error {
	pluginID := c.Param("pluginId")
	if pluginID == "" {
		s.logger.Error("plugin id is required")
		return c.JSON(http.StatusBadRequest, NewErrorResponse("plugin id is required"))
	}

	var plugin types.PluginUpdateDto
	if err := c.Bind(&plugin); err != nil {
		s.logger.WithError(err).Error("Failed to parse request")
		return c.JSON(http.StatusBadRequest, NewErrorResponse("failed to parse request"))
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
		s.logger.WithError(err).Error("Failed to get categories")
		return c.JSON(http.StatusInternalServerError, NewErrorResponse("failed to get categories"))
	}

	return c.JSON(http.StatusOK, categories)
}

func (s *Server) GetTags(c echo.Context) error {
	tags, err := s.db.FindTags(c.Request().Context())
	if err != nil {
		s.logger.WithError(err).Error("Failed to get tags")
		return c.JSON(http.StatusInternalServerError, NewErrorResponse("failed to get tags"))
	}
	return c.JSON(http.StatusOK, tags)
}

func (s *Server) AttachPluginTag(c echo.Context) error {
	pluginID := c.Param("pluginId")
	if pluginID == "" {
		return c.JSON(http.StatusBadRequest, NewErrorResponse("plugin id is required"))
	}
	_, err := s.db.FindPluginById(c.Request().Context(), nil, ptypes.PluginID(pluginID))
	if err != nil {
		s.logger.WithError(err).Error("Failed to find plugin")
		if errors.Is(err, pgx.ErrNoRows) {
			return c.JSON(http.StatusNotFound, NewErrorResponse("plugin not found"))
		}
		return c.JSON(http.StatusInternalServerError, NewErrorResponse("failed to find plugin"))
	}

	var createTagDto types.CreateTagDto
	if err := c.Bind(&createTagDto); err != nil {
		s.logger.WithError(err).Error("Failed to parse request")
		return c.JSON(http.StatusBadRequest, NewErrorResponse("failed to parse request"))
	}
	if err := c.Validate(&createTagDto); err != nil {
		s.logger.Error(err)
		return c.JSON(http.StatusBadRequest, NewErrorResponse(err.Error()))
	}

	var tag *types.Tag
	tag, err = s.db.FindTagByName(c.Request().Context(), createTagDto.Name)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			tag, err = s.db.CreateTag(c.Request().Context(), createTagDto)
			if err != nil {
				s.logger.WithError(err).Error("Failed to create tag")
				return c.JSON(http.StatusInternalServerError, NewErrorResponse("failed to create tag"))
			}
		} else {
			s.logger.WithError(err).Error("Failed to check for existing tag")
			return c.JSON(http.StatusInternalServerError, NewErrorResponse("failed to check for existing tag"))
		}
	}

	updatedPlugin, err := s.db.AttachTagToPlugin(c.Request().Context(), ptypes.PluginID(pluginID), tag.ID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to attach tag")
		return c.JSON(http.StatusInternalServerError, NewErrorResponse("failed to attach tag"))
	}

	return c.JSON(http.StatusOK, updatedPlugin)
}

func (s *Server) DetachPluginTag(c echo.Context) error {
	pluginID := c.Param("pluginId")
	if pluginID == "" {
		return c.JSON(http.StatusBadRequest, NewErrorResponse("plugin id is required"))
	}
	_, err := s.db.FindPluginById(c.Request().Context(), nil, ptypes.PluginID(pluginID))
	if err != nil {
		s.logger.WithError(err).Error("Failed to find plugin")
		if errors.Is(err, pgx.ErrNoRows) {
			return c.JSON(http.StatusNotFound, NewErrorResponse("plugin not found"))
		}
		return c.JSON(http.StatusInternalServerError, NewErrorResponse("failed to find plugin"))
	}

	tagID := c.Param("tagId")
	if tagID == "" {
		return c.JSON(http.StatusBadRequest, NewErrorResponse("tag id is required"))
	}
	tag, err := s.db.FindTagById(c.Request().Context(), tagID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to find tag")
		if errors.Is(err, pgx.ErrNoRows) {
			return c.JSON(http.StatusNotFound, NewErrorResponse("tag not found"))
		}
		return c.JSON(http.StatusInternalServerError, NewErrorResponse("failed to find tag"))
	}

	updatedPlugin, err := s.db.DetachTagFromPlugin(c.Request().Context(), ptypes.PluginID(pluginID), tag.ID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to detach tag")
		return c.JSON(http.StatusInternalServerError, NewErrorResponse("failed to detach tag"))
	}

	return c.JSON(http.StatusOK, updatedPlugin)
}

func (s *Server) GetPluginPolicyTransactionHistory(c echo.Context) error {
	policyID := c.Param("policyId")

	if policyID == "" {
		err := fmt.Errorf("policy ID is required")
		return c.JSON(http.StatusBadRequest, NewErrorResponse(err.Error()))
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
		s.logger.WithError(err).Error("Failed to get policy history")
		return c.JSON(http.StatusInternalServerError, NewErrorResponse("failed to get policy history"))
	}

	return c.JSON(http.StatusOK, policyHistory)
}

func (s *Server) CreateReview(c echo.Context) error {
	var review types.ReviewCreateDto
	if err := c.Bind(&review); err != nil {
		s.logger.WithError(err).Error("Failed to parse request")
		return c.JSON(http.StatusBadRequest, NewErrorResponse("failed to parse request"))
	}

	if err := c.Validate(&review); err != nil {
		return c.JSON(http.StatusBadRequest, NewErrorResponse(err.Error()))
	}

	// If allowing HTML, sanitize with bluemonday:
	p := bluemonday.UGCPolicy()
	review.Comment = p.Sanitize(review.Comment)

	pluginID := c.Param("pluginId")
	if pluginID == "" {
		err := fmt.Errorf("plugin id is required")
		s.logger.Error(err)
		return c.JSON(http.StatusBadRequest, NewErrorResponse(err.Error()))
	}

	created, err := s.pluginService.CreatePluginReviewWithRating(c.Request().Context(), review, pluginID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to create review")
		return c.JSON(http.StatusInternalServerError, NewErrorResponse("failed to create review"))
	}

	return c.JSON(http.StatusOK, created)
}

func (s *Server) GetReviews(c echo.Context) error {
	pluginId := c.Param("pluginId")
	if pluginId == "" {
		err := fmt.Errorf("plugin id is required")
		s.logger.Error(err)
		return c.JSON(http.StatusBadRequest, NewErrorResponse(err.Error()))
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

	allowedSortFields := []string{"created_at", "rating", "updated_at"}
	if sort != "" && !common.IsValidSortField(sort, allowedSortFields) {
		return c.JSON(http.StatusBadRequest, NewErrorResponse("invalid sort parameter"))
	}

	reviews, err := s.db.FindReviews(c.Request().Context(), pluginId, skip, take, sort)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get reviews")
		return c.JSON(http.StatusInternalServerError, NewErrorResponse("failed to get reviews"))
	}

	return c.JSON(http.StatusOK, reviews)
}
