package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/labstack/echo/v4"
	"github.com/microcosm-cc/bluemonday"
	"github.com/vultisig/verifier/internal/conv"
	"github.com/vultisig/verifier/tx_indexer/pkg/storage"

	"github.com/vultisig/verifier/common"

	"github.com/vultisig/recipes/chain"
	"github.com/vultisig/recipes/engine"

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

	// Get policy from database
	policy, err := s.db.GetPluginPolicy(c.Request().Context(), req.PolicyID)
	if err != nil {
		return fmt.Errorf("failed to get policy from database: %w", err)
	}

	// Validate policy matches plugin
	if policy.PluginID != ptypes.PluginID(req.PluginID) {
		return fmt.Errorf("policy plugin ID mismatch")
	}

	recipe, err := policy.GetRecipe()
	if err != nil {
		return fmt.Errorf("failed to get recipe: %w", err)
	}

	eng := engine.NewEngine()

	for i, keysignMessage := range req.Messages {
		messageChain, err := chain.GetChain(strings.ToLower(keysignMessage.Chain.String()))
		if err != nil {
			return fmt.Errorf("failed to get chain: %w", err)
		}

		decodedTx, err := messageChain.ParseTransaction(keysignMessage.Message)
		if err != nil {
			return fmt.Errorf("failed to parse transaction: %w", err)
		}

		var txToTrackID uuid.UUID
		if s.txIndexerService != nil {
			txToTrack, err := s.txIndexerService.CreateTx(c.Request().Context(), storage.CreateTxDto{
				PluginID:      ptypes.PluginID(req.PluginID),
				ChainID:       keysignMessage.Chain,
				PolicyID:      policy.ID,
				FromPublicKey: req.PublicKey,
				ProposedTxHex: keysignMessage.Message,
			})
			if err != nil {
				return fmt.Errorf("s.txIndexerService.CreateTx(: %w", err)
			}
			txToTrackID = txToTrack.ID
			req.Messages[i].TxIndexerID = txToTrack.ID.String()
		}

		transactionAllowed, _, err := eng.Evaluate(recipe, messageChain, decodedTx)
		if err != nil {
			return fmt.Errorf("failed to evaluate policy: %w", err)
		}

		if !transactionAllowed {
			return fmt.Errorf("transaction %s on %s not allowed by policy", keysignMessage.Hash, keysignMessage.Chain)
		}
		if s.txIndexerService != nil {
			err = s.txIndexerService.SetStatus(c.Request().Context(), txToTrackID, storage.TxVerified)
			if err != nil {
				return fmt.Errorf("tx_id=%s, failed to set transaction status to verified: %w", txToTrackID, err)
			}
		}
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

	ti, err := s.asynqClient.EnqueueContext(c.Request().Context(),
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
	skip, take, err := conv.PageParamsFromCtx(c, 0, 20)
	if err != nil {
		s.logger.WithError(err).Error("conv.PageParamsFromCtx")
		return c.JSON(http.StatusBadRequest, NewErrorResponse(
			http.StatusBadRequest,
			"invalid pagination parameters",
			err.Error(),
		))
	}

	sort := c.QueryParam("sort")

	filters := types.PluginFilters{
		Term:       common.GetQueryParam(c, "term"),
		TagID:      common.GetUUIDParam(c, "tag_id"),
		CategoryID: common.GetUUIDParam(c, "category_id"),
	}

	plugins, err := s.db.FindPlugins(c.Request().Context(), filters, int(take), int(skip), sort)

	if err != nil {
		s.logger.WithError(err).Error("Failed to get plugins")
		return c.JSON(http.StatusInternalServerError, NewErrorResponse(http.StatusInternalServerError, "failed to get plugins", err.Error()))
	}

	return c.JSON(http.StatusOK, plugins)
}

func (s *Server) GetPlugin(c echo.Context) error {
	pluginID := c.Param("pluginId")
	if pluginID == "" {
		s.logger.Error("plugin id is required")
		return c.JSON(http.StatusBadRequest, NewErrorResponse(http.StatusBadRequest, "plugin id is required", ""))
	}

	plugin, err := s.pluginService.GetPluginWithRating(c.Request().Context(), pluginID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get plugin")
		return c.JSON(http.StatusInternalServerError, NewErrorResponse(http.StatusInternalServerError, "failed to get plugin", err.Error()))
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
		return c.JSON(http.StatusInternalServerError, NewErrorResponse(http.StatusInternalServerError, "failed to get tags", err.Error()))
	}
	return c.JSON(http.StatusOK, tags)
}

func (s *Server) GetPluginPolicyTransactionHistory(c echo.Context) error {
	policyID := c.Param("policyId")
	if policyID == "" {
		err := fmt.Errorf("policy ID is required")
		return c.JSON(http.StatusBadRequest, NewErrorResponse(http.StatusBadRequest, err.Error(), ""))
	}
	policyUUID, err := uuid.Parse(policyID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, NewErrorResponse(
			http.StatusBadRequest,
			"'policyId' invalid uuid",
			err.Error(),
		))
	}

	skip, take, err := conv.PageParamsFromCtx(c, 0, 20)
	if err != nil {
		return c.JSON(http.StatusBadRequest, NewErrorResponse(
			http.StatusBadRequest,
			"invalid pagination parameters",
			err.Error(),
		))
	}

	txs, totalCount, err := s.txIndexerService.GetByPolicyID(c.Request().Context(), policyUUID, skip, take)
	if err != nil {
		s.logger.WithError(err).Errorf("s.txIndexerService.GetByPolicyID: %s", policyID)
		return c.JSON(http.StatusInternalServerError, NewErrorResponse(
			http.StatusInternalServerError,
			"failed to get txs by policy ID",
			err.Error(),
		))
	}

	return c.JSON(http.StatusOK, types.TransactionHistoryPaginatedList{
		History:    txs,
		TotalCount: totalCount,
	})
}

func (s *Server) CreateReview(c echo.Context) error {
	var review types.ReviewCreateDto
	if err := c.Bind(&review); err != nil {
		s.logger.WithError(err).Error("Failed to parse request")
		return c.JSON(http.StatusBadRequest, NewErrorResponse(http.StatusBadRequest, "failed to parse request", err.Error()))
	}

	if err := c.Validate(&review); err != nil {
		s.logger.WithError(err).Error("CreateReview: Request validation failed")
		return c.JSON(http.StatusBadRequest, NewErrorResponse(http.StatusBadRequest, err.Error(), err.Error()))
	}

	// If allowing HTML, sanitize with bluemonday:
	p := bluemonday.UGCPolicy()
	review.Comment = p.Sanitize(review.Comment)

	pluginID := c.Param("pluginId")
	if pluginID == "" {
		s.logger.Error("CreateReview: Missing plugin ID in URL parameters")
		return c.JSON(http.StatusBadRequest, NewErrorResponse(http.StatusBadRequest, "plugin id is required", "plugin id is required"))
	}

	created, err := s.pluginService.CreatePluginReviewWithRating(c.Request().Context(), review, pluginID)
	if err != nil {
		s.logger.WithError(err).Errorf("CreateReview: Plugin service failed to create review for plugin %s", pluginID)
		return c.JSON(http.StatusInternalServerError, NewErrorResponse(http.StatusInternalServerError, "failed to create review", err.Error()))
	}

	return c.JSON(http.StatusOK, created)
}

func (s *Server) GetReviews(c echo.Context) error {
	pluginId := c.Param("pluginId")
	if pluginId == "" {
		err := fmt.Errorf("plugin id is required")
		s.logger.Error(err)
		return c.JSON(http.StatusBadRequest, NewErrorResponse(http.StatusBadRequest, err.Error(), ""))
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
		return c.JSON(http.StatusBadRequest, NewErrorResponse(http.StatusBadRequest, "invalid sort parameter", ""))
	}

	reviews, err := s.db.FindReviews(c.Request().Context(), pluginId, skip, take, sort)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get reviews")
		return c.JSON(http.StatusInternalServerError, NewErrorResponse(http.StatusInternalServerError, "failed to get reviews", ""))
	}

	return c.JSON(http.StatusOK, reviews)
}

// GET /plugins/:pluginId/recipe-specification
func (s *Server) GetPluginRecipeSpecification(c echo.Context) error {
	pluginID := c.Param("pluginId")
	if pluginID == "" {
		return c.JSON(http.StatusBadRequest, NewErrorResponse(http.StatusBadRequest, "plugin id is required", ""))
	}

	s.logger.Debugf("[GetPluginRecipeSpecification] Getting recipe spec for pluginID=%s\n", pluginID)

	recipeSpec, err := s.pluginService.GetPluginRecipeSpecification(c.Request().Context(), pluginID)
	if err != nil {
		s.logger.WithError(err).Error("[GetPluginRecipeSpecification] Failed to get plugin recipe specification")
		return c.JSON(http.StatusInternalServerError, NewErrorResponse(http.StatusInternalServerError, "failed to get recipe specification", ""))
	}

	s.logger.Debugf("[GetPluginRecipeSpecification] Successfully got recipe spec for plugin %s\n", pluginID)
	return c.JSON(http.StatusOK, recipeSpec)
}
