package api

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/btcsuite/btcd/btcutil/psbt"
	ecommon "github.com/ethereum/go-ethereum/common"
	etypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/labstack/echo/v4"
	"github.com/microcosm-cc/bluemonday"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/vultisig/recipes/engine"
	"github.com/vultisig/recipes/ethereum"
	rtypes "github.com/vultisig/recipes/types"
	"github.com/vultisig/verifier/internal/conv"
	"github.com/vultisig/verifier/internal/types"
	"github.com/vultisig/verifier/plugin/tasks"
	"github.com/vultisig/verifier/plugin/tx_indexer/pkg/storage"
	ptypes "github.com/vultisig/verifier/types"
	"github.com/vultisig/vultisig-go/common"
)

func (s *Server) SignPluginMessages(c echo.Context) error {
	s.logger.Debug("PLUGIN SERVER: SIGN MESSAGES")

	var req ptypes.PluginKeysignRequest
	if err := c.Bind(&req); err != nil {
		return fmt.Errorf("fail to parse request, err: %w", err)
	}

	// Get policy from database
	if req.PluginID == ptypes.PluginVultisigFees_feee.String() {
		s.logger.Debug("SIGN FEE PLUGIN MESSAGES")
		return s.validateAndSign(c, &req, types.FeeDefaultPolicy, uuid.New())
	} else {
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
			return fmt.Errorf("failed to unpack recipe: %w", err)
		}

		if recipe.RateLimitWindow != nil && recipe.MaxTxsPerWindow != nil {
			txs, er := s.txIndexerService.GetTxsInTimeRange(
				c.Request().Context(),
				policy.ID,
				time.Now().Add(time.Duration(-recipe.GetRateLimitWindow())*time.Second),
				time.Now(),
			)
			if er != nil {
				return fmt.Errorf("failed to get data from tx indexer: %w", er)
			}
			if uint32(len(txs)) >= recipe.GetMaxTxsPerWindow() {
				return fmt.Errorf(
					"policy not allowed to execute more txs in currrent time window: "+
						"policy_id=%s, txs=%d, max_txs=%d, min_exec_window=%d",
					policy.ID.String(),
					len(txs),
					recipe.GetMaxTxsPerWindow(),
					recipe.GetRateLimitWindow(),
				)
			}
		}
		return s.validateAndSign(c, &req, recipe, policy.ID)
	}
}

func (s *Server) validateAndSign(c echo.Context, req *ptypes.PluginKeysignRequest, recipe *rtypes.Policy, policyID uuid.UUID) error {
	if len(req.Messages) == 0 {
		return errors.New("no messages to sign")
	}

	firstKeysignMessage := req.Messages[0]

	// actually doesn't make a lot of sense to validate that,
	// because even if req.Messages[](hashes proposed to sign) and req.Transaction(proposed tx object) are different,
	// we are always appending signatures to req.Transaction and if it were for different hashes,
	// signature would be wrong, and this tx would be rejected by the blockchain node anyway
	//
	// consider it's not safety check, but rather a sanity check
	var txBytesEvaluate []byte
	switch {
	case firstKeysignMessage.Chain.IsEvm():
		if len(req.Messages) != 1 {
			return errors.New("plugins must sign exactly 1 message for evm")
		}

		b, er := base64.StdEncoding.DecodeString(req.Transaction)
		if er != nil {
			return fmt.Errorf("failed to decode b64 proposed tx: %w", er)
		}
		txBytesEvaluate = b

		evmID, er := firstKeysignMessage.Chain.EvmID()
		if er != nil {
			return fmt.Errorf("evm chain id not found: %s", firstKeysignMessage.Chain.String())
		}

		txData, er := ethereum.DecodeUnsignedPayload(b)
		if er != nil {
			return fmt.Errorf("failed to decode evm payload: %w", er)
		}

		kmBytes, er := base64.StdEncoding.DecodeString(firstKeysignMessage.Message)
		if er != nil {
			return fmt.Errorf("failed to decode b64 proposed tx: %w", er)
		}

		hashToSignFromTxObj := etypes.LatestSignerForChainID(evmID).Hash(etypes.NewTx(txData))
		hashToSign := ecommon.BytesToHash(kmBytes)
		if hashToSignFromTxObj.Cmp(hashToSign) != 0 {
			// req.Transaction — full tx to unpack and assert ERC20 args against policy
			// keysignMessage.Message — is ECDSA hash to sign
			//
			// We must validate that plugin not cheating and
			// hash from req.Transaction is the same as keysignMessage.Message,
			return fmt.Errorf(
				"hashToSign(%s) must be the same as computed hash from request.Transaction(%s)",
				hashToSign.Hex(),
				hashToSignFromTxObj.Hex(),
			)
		}
	case firstKeysignMessage.Chain == common.Bitcoin:
		b, er := psbt.NewFromRawBytes(strings.NewReader(req.Transaction), true)
		if er != nil {
			return fmt.Errorf("failed to decode psbt: %w", er)
		}

		var buf bytes.Buffer
		er = b.UnsignedTx.Serialize(&buf)
		if er != nil {
			return fmt.Errorf("failed to serialize psbt: %w", er)
		}

		txBytesEvaluate = buf.Bytes()
	case firstKeysignMessage.Chain == common.XRP:
		b, er := base64.StdEncoding.DecodeString(req.Transaction)
		if er != nil {
			return fmt.Errorf("failed to decode base64 XRP transaction: %w", er)
		}
		txBytesEvaluate = b
	case firstKeysignMessage.Chain == common.Solana:
		b, er := base64.StdEncoding.DecodeString(req.Transaction)
		if er != nil {
			return fmt.Errorf("failed to decode b64 proposed Solana tx: %w", er)
		}
		txBytesEvaluate = b
	default:
		return fmt.Errorf("failed to decode transaction, chain %s not supported", firstKeysignMessage.Chain)
	}

	ngn, err := engine.NewEngine()
	if err != nil {
		return fmt.Errorf("failed to create engine: %w", err)
	}
	//TODO: fee plugin priority for testing purposes
	if req.PluginID == ptypes.PluginVultisigFees_feee.String() {
		_, err = ngn.Evaluate(types.FeeDefaultPolicy, firstKeysignMessage.Chain, txBytesEvaluate)
		if err != nil {
			return fmt.Errorf("tx not allowed to execute: %w", err)
		}
	} else {
		_, err = ngn.Evaluate(recipe, firstKeysignMessage.Chain, txBytesEvaluate)
		if err != nil {
			return fmt.Errorf("tx not allowed to execute: %w", err)
		}
	}

	txToTrack, err := s.txIndexerService.CreateTx(c.Request().Context(), storage.CreateTxDto{
		PluginID:      ptypes.PluginID(req.PluginID),
		ChainID:       firstKeysignMessage.Chain,
		PolicyID:      policyID,
		FromPublicKey: req.PublicKey,
		ProposedTxHex: req.Transaction,
	})
	if err != nil {
		return fmt.Errorf("failed to create tx for tracking: %w", err)
	}
	req.Messages[0].TxIndexerID = txToTrack.ID.String()

	err = s.txIndexerService.SetStatus(c.Request().Context(), txToTrack.ID, storage.TxVerified)
	if err != nil {
		return fmt.Errorf("tx_id=%s, failed to set transaction status to verified: %w", txToTrack.ID, err)
	}

	// Reuse existing signing logic
	result, err := s.redis.Get(c.Request().Context(), req.SessionID)
	if err == nil && result != "" {
		return c.NoContent(http.StatusOK)
	}

	if err := s.redis.Set(c.Request().Context(), req.SessionID, req.SessionID, 30*time.Minute); err != nil {
		s.logger.WithError(err).Error("fail to set session")
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
		s.logger.WithError(err).Error("fail to parse pagination parameters")
		return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage(msgInvalidPagination))
	}
	sort := c.QueryParam("sort")
	filters := types.PluginFilters{
		Term:       common.GetQueryParam(c, "term"),
		TagID:      common.GetUUIDParam(c, "tag_id"),
		CategoryID: common.GetQueryParam(c, "category_id"),
	}

	plugins, err := s.db.FindPlugins(c.Request().Context(), filters, int(take), int(skip), sort)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get plugins")
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgGetPluginsFailed))
	}

	return c.JSON(http.StatusOK, plugins)
}

func (s *Server) GetPlugin(c echo.Context) error {
	pluginID := c.Param("pluginId")
	if pluginID == "" {
		s.logger.Error("plugin id is required")
		return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage(msgRequiredPluginID))
	}

	plugin, err := s.pluginService.GetPluginWithRating(c.Request().Context(), pluginID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get plugin")
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgGetPluginFailed))
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
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgGetTagsFailed))
	}
	return c.JSON(http.StatusOK, tags)
}

func (s *Server) GetPluginPolicyTransactionHistory(c echo.Context) error {
	policyID := c.Param("policyId")
	if policyID == "" {
		return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage(msgRequiredPolicyID))
	}
	policyUUID, err := uuid.Parse(policyID)
	if err != nil {
		return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage(msgInvalidPolicyID))
	}

	skip, take, err := conv.PageParamsFromCtx(c, 0, 20)
	if err != nil {
		return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage(msgInvalidPagination))
	}

	if take > 100 {
		take = 100
	}
	publicKey, ok := c.Get("vault_public_key").(string)
	if !ok {
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgVaultPublicKeyGetFailed))
	}
	oldPolicy, err := s.policyService.GetPluginPolicy(c.Request().Context(), policyUUID)
	if err != nil {
		s.logger.WithError(err).Errorf("failed to get plugin policy, id:%s", policyUUID)
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgPolicyGetFailed))
	}
	if oldPolicy.PublicKey != publicKey {
		return c.JSON(http.StatusForbidden, NewErrorResponseWithMessage(msgPublicKeyMismatch))
	}

	txs, totalCount, err := s.txIndexerService.GetByPolicyID(c.Request().Context(), policyUUID, skip, take)
	if err != nil {
		s.logger.WithError(err).Errorf("s.txIndexerService.GetByPolicyID: %s", policyID)
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgGetTxsByPolicyIDFailed))
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
		return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage(msgRequestParseFailed))
	}

	if err := c.Validate(&review); err != nil {
		s.logger.WithError(err).Error("Request validation failed")
		return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage(msgInvalidReview))
	}

	// If allowing HTML, sanitize with bluemonday:
	p := bluemonday.UGCPolicy()
	review.Comment = p.Sanitize(review.Comment)

	pluginID := c.Param("pluginId")
	if pluginID == "" {
		return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage(msgRequiredPluginID))
	}

	created, err := s.pluginService.CreatePluginReviewWithRating(c.Request().Context(), review, pluginID)
	if err != nil {
		s.logger.WithError(err).Errorf("Plugin service failed to create review for plugin %s", pluginID)
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgCreateReviewFailed))
	}

	return c.JSON(http.StatusOK, created)
}

func (s *Server) GetReviews(c echo.Context) error {
	pluginId := c.Param("pluginId")
	if pluginId == "" {
		return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage(msgRequiredPluginID))
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
		return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage(msgInvalidSort))
	}

	reviews, err := s.db.FindReviews(c.Request().Context(), pluginId, skip, take, sort)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get reviews")
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgGetReviewsFailed))
	}

	return c.JSON(http.StatusOK, reviews)
}

func (s *Server) GetPluginAvgRating(c echo.Context) error {
	pluginID := c.Param("pluginId")
	if pluginID == "" {
		return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage(msgRequiredPluginID))
	}

	avgRating, err := s.db.FindAvgRatingByPluginID(c.Request().Context(), pluginID)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get average rating")
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgGetAvgRatingFailed))
	}

	return c.JSON(http.StatusOK, avgRating)
}

func (s *Server) GetPluginRecipeSpecification(c echo.Context) error {
	pluginID := c.Param("pluginId")
	if pluginID == "" {
		return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage(msgRequiredPluginID))
	}
	recipeSpec, err := s.pluginService.GetPluginRecipeSpecification(c.Request().Context(), pluginID)
	if err != nil {
		s.logger.WithError(err).Error("failed to get plugin recipe specification")
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgGetRecipeSpecFailed))
	}

	return c.JSON(http.StatusOK, recipeSpec)
}

func (s *Server) GetPluginRecipeSpecificationSuggest(c echo.Context) error {
	pluginID := c.Param("pluginId")
	if pluginID == "" {
		return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage(msgRequiredPluginID))
	}

	type reqBody struct {
		Configuration map[string]any `json:"configuration"`
	}
	var req reqBody
	err := c.Bind(&req)
	if err != nil {
		s.logger.WithError(err).Error("failed to parse request")
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgRequestParseFailed))
	}

	recipeSpec, err := s.pluginService.GetPluginRecipeSpecificationSuggest(
		c.Request().Context(),
		pluginID,
		req.Configuration,
	)
	if err != nil {
		s.logger.WithError(err).Error("failed to get plugin recipe suggest")
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgGetRecipeSuggestFailed))
	}

	b, err := protojson.Marshal(recipeSpec)
	if err != nil {
		s.logger.WithError(err).Error("failed to proto marshal")
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgProtoMarshalFailed))
	}

	var res map[string]any
	err = json.Unmarshal(b, &res)
	if err != nil {
		s.logger.WithError(err).Error("failed to json marshal")
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgJSONMarshalFailed))
	}

	return c.JSON(http.StatusOK, res)
}

func (s *Server) GetPluginRecipeFunctions(c echo.Context) error {
	pluginID := c.Param("pluginId")
	if pluginID == "" {
		return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage(msgRequiredPluginID))
	}
	recipeFuncs, err := s.pluginService.GetPluginRecipeFunctions(c.Request().Context(), pluginID)
	if err != nil {
		s.logger.WithError(err).Error(msgGetRecipeFunctionsFailed)
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgGetRecipeFunctionsFailed))
	}

	return c.JSON(http.StatusOK, recipeFuncs)
}

func (s *Server) GetPluginInstallationsCountByID(c echo.Context) error {
	pluginID := c.Param("pluginId")
	if pluginID == "" {
		return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage(msgRequiredPluginID))
	}

	count, err := s.policyService.GetPluginInstallationsCount(c.Request().Context(), ptypes.PluginID(pluginID))
	if err != nil {
		s.logger.WithError(err).Errorf("Failed to get installation count for pluginId: %s", pluginID)
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgPluginInstallationCountFailed))
	}

	return c.JSON(http.StatusOK, count)
}
