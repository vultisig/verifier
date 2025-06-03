package api

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
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
	policyBytes, err := base64.StdEncoding.DecodeString(policy.Recipe)
	if err != nil {
		return fmt.Errorf("failed to decode policy recipe: %w", err)
	}

	if err := protojson.Unmarshal(policyBytes, &recipe); err != nil {
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
			s.logger.WithError(err).Errorf("skipping rule %s: invalid resource path %s", rule.GetId(), resourcePathString)
			continue
		}

		if resourcePath.ChainId != "ethereum" {
			s.logger.Errorf("skipping rule %s: target chain %s is not 'ethereum'", rule.GetId(), resourcePath.ChainId)
			continue
		}

		protocol, err := ethChain.GetProtocol(resourcePath.ProtocolId)
		if err != nil {
			s.logger.WithError(err).Errorf("skipping rule %s: could not get protocol for asset '%s'", rule.GetId(), resourcePath.ProtocolId)
			continue
		}

		policyMatcher := &rtypes.PolicyFunctionMatcher{
			FunctionID:  resourcePath.FunctionId,
			Constraints: rule.GetParameterConstraints(), // Use generated getter
		}

		matches, _, err := protocol.MatchFunctionCall(decodedTx, policyMatcher)
		if err != nil {
			s.logger.WithError(err).Errorf("error during transaction matching for rule %s, function %s", rule.GetId(), resourcePath.FunctionId)
			continue
		}

		if matches {
			transactionAllowedByPolicy = true
			break
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

func (s *Server) GetCategories(c echo.Context) error {
	resp := []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}{
		{
			ID:   string(types.PluginCategoryAIAgent),
			Name: types.PluginCategoryAIAgent.String(),
		},
		{
			ID:   string(types.PluginCategoryPlugin),
			Name: types.PluginCategoryPlugin.String(),
		},
	}

	return c.JSON(http.StatusOK, resp)
}

func (s *Server) GetTags(c echo.Context) error {
	tags, err := s.db.FindTags(c.Request().Context())
	if err != nil {
		s.logger.WithError(err).Error("Failed to get tags")
		return c.JSON(http.StatusInternalServerError, NewErrorResponse("failed to get tags"))
	}
	return c.JSON(http.StatusOK, tags)
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

// GET /plugins/:pluginId/recipe-specification
func (s *Server) GetPluginRecipeSpecification(c echo.Context) error {
	pluginID := c.Param("pluginId")
	if pluginID == "" {
		return c.JSON(http.StatusBadRequest, NewErrorResponse("plugin id is required"))
	}

	s.logger.Debugf("[GetPluginRecipeSpecification] Getting recipe spec for pluginID=%s\n", pluginID)

	recipeSpec, err := s.pluginService.GetPluginRecipeSpecification(c.Request().Context(), pluginID)
	if err != nil {
		s.logger.WithError(err).Error("[GetPluginRecipeSpecification] Failed to get plugin recipe specification")
		return c.JSON(http.StatusInternalServerError, NewErrorResponse("failed to get recipe specification"))
	}

	s.logger.Debugf("[GetPluginRecipeSpecification] Successfully got recipe spec for plugin %s\n", pluginID)
	return c.JSON(http.StatusOK, recipeSpec)
}
