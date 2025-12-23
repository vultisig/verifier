package api

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
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
	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"
	"github.com/microcosm-cc/bluemonday"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/vultisig/recipes/chain/evm/ethereum"
	"github.com/vultisig/recipes/engine"
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
		errMsg := "fail to parse request"
		return s.badRequest(c, errMsg, err)
	}

	// Get policy from database
	if req.PluginID == ptypes.PluginVultisigFees_feee.String() {
		s.logger.Debug("SIGN FEE PLUGIN MESSAGES")
		return s.validateAndSign(c, &req, types.FeeDefaultPolicy, uuid.New())
	} else {
		var (
			isTrialActive bool
			err           error
		)
		err = s.db.WithTransaction(c.Request().Context(), func(ctx context.Context, tx pgx.Tx) error {
			isTrialActive, _, err = s.db.IsTrialActive(ctx, tx, req.PublicKey)
			return err
		})
		if err != nil {
			errMsg := "failed to check trial info"
			return s.internal(c, errMsg, err)
		}

		if !isTrialActive {
			filePathName := common.GetVaultBackupFilename(req.PublicKey, ptypes.PluginVultisigFees_feee.String())
			exist, err := s.vaultStorage.Exist(filePathName)
			if err != nil {
				errMsg := "failed to check vault existence"
				return s.internal(c, errMsg, err)
			}
			if !exist {
				return c.JSON(http.StatusForbidden, NewErrorResponseWithMessage(msgAccessDeniedBilling))
			}
		}

		policy, err := s.db.GetPluginPolicy(c.Request().Context(), req.PolicyID)
		if err != nil {
			errMsg := "failed to get policy from database"
			return s.internal(c, errMsg, err)
		}

		// Validate policy matches plugin
		if policy.PluginID != ptypes.PluginID(req.PluginID) {
			errMsg := "policy plugin ID mismatch"
			return s.badRequest(c, errMsg, nil)
		}

		recipe, err := policy.GetRecipe()
		if err != nil {
			errMsg := "failed to unpack recipe"
			return s.internal(c, errMsg, err)
		}

		if recipe.RateLimitWindow != nil && recipe.MaxTxsPerWindow != nil {
			txs, err := s.txIndexerService.GetTxsInTimeRange(
				c.Request().Context(),
				policy.ID,
				time.Now().Add(time.Duration(-recipe.GetRateLimitWindow())*time.Second),
				time.Now(),
			)
			if err != nil {
				errMsg := "failed to get data from tx indexer"
				return s.internal(c, errMsg, err)
			}
			if uint32(len(txs)) >= recipe.GetMaxTxsPerWindow() {
				errMsg := fmt.Sprintf(
					"policy not allowed to execute more txs in currrent time window: "+
						"policy_id=%s, txs=%d, max_txs=%d, min_exec_window=%d",
					policy.ID.String(),
					len(txs),
					recipe.GetMaxTxsPerWindow(),
					recipe.GetRateLimitWindow(),
				)
				s.logger.Error(errMsg)
				return c.JSON(http.StatusTooManyRequests, NewErrorResponseWithMessage(errMsg))

			}
		}
		return s.validateAndSign(c, &req, recipe, policy.ID)
	}
}

func (s *Server) validateAndSign(c echo.Context, req *ptypes.PluginKeysignRequest, recipe *rtypes.Policy, policyID uuid.UUID) error {
	if len(req.Messages) == 0 {
		return s.badRequest(c, "no messages to sign", nil)
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
			errMsg := "plugins must sign exactly 1 message for evm"
			return s.badRequest(c, errMsg, nil)
		}

		b, err := base64.StdEncoding.DecodeString(req.Transaction)
		if err != nil {
			errMsg := "transaction must be base64"
			return s.badRequest(c, errMsg, err)
		}
		txBytesEvaluate = b

		evmID, err := firstKeysignMessage.Chain.EvmID()
		if err != nil {
			errMsg := fmt.Sprintf("evm chain id not found: %s", firstKeysignMessage.Chain.String())
			return s.badRequest(c, errMsg, err)
		}

		txData, err := ethereum.DecodeUnsignedPayload(b)
		if err != nil {
			errMsg := "failed to decode evm payload"
			return s.badRequest(c, errMsg, err)
		}

		kmBytes, err := base64.StdEncoding.DecodeString(firstKeysignMessage.Message)
		if err != nil {
			errMsg := "failed to decode b64 proposed tx"
			return s.badRequest(c, errMsg, err)
		}

		hashToSignFromTxObj := etypes.LatestSignerForChainID(evmID).Hash(etypes.NewTx(txData))
		hashToSign := ecommon.BytesToHash(kmBytes)
		if hashToSignFromTxObj.Cmp(hashToSign) != 0 {
			// req.Transaction — full tx to unpack and assert ERC20 args against policy
			// keysignMessage.Message — is ECDSA hash to sign
			//
			// We must validate that plugin not cheating and
			// hash from req.Transaction is the same as keysignMessage.Message,
			errMsg := fmt.Sprintf("hashToSign(%s) must be the same as computed hash from request.Transaction(%s)",
				hashToSign.Hex(),
				hashToSignFromTxObj.Hex())
			return s.badRequest(c, errMsg, nil)
		}
	case firstKeysignMessage.Chain == common.Bitcoin:
		b, err := psbt.NewFromRawBytes(strings.NewReader(req.Transaction), true)
		if err != nil {
			errMsg := "failed to decode psbt"
			return s.badRequest(c, errMsg, err)
		}

		var buf bytes.Buffer
		err = b.UnsignedTx.Serialize(&buf)
		if err != nil {
			errMsg := "failed to serialize psbt"
			return s.badRequest(c, errMsg, err)
		}

		txBytesEvaluate = buf.Bytes()
	case firstKeysignMessage.Chain == common.XRP:
		b, err := base64.StdEncoding.DecodeString(req.Transaction)
		if err != nil {
			errMsg := "failed to decode base64 XRP transaction"
			return s.badRequest(c, errMsg, err)
		}
		txBytesEvaluate = b
	case firstKeysignMessage.Chain == common.Solana:
		b, err := base64.StdEncoding.DecodeString(req.Transaction)
		if err != nil {
			errMsg := "failed to decode base64 Solana tx"
			return s.badRequest(c, errMsg, err)
		}
		txBytesEvaluate = b
	default:
		errMsg := fmt.Sprintf("failed to decode transaction, chain %s not supported", firstKeysignMessage.Chain)
		return s.badRequest(c, errMsg, nil)
	}

	ngn, err := engine.NewEngine()
	if err != nil {
		errMsg := "failed to create engine"
		return s.internal(c, errMsg, err)
	}
	//TODO: fee plugin priority for testing purposes
	if req.PluginID == ptypes.PluginVultisigFees_feee.String() {
		_, err = ngn.Evaluate(types.FeeDefaultPolicy, firstKeysignMessage.Chain, txBytesEvaluate)
		if err != nil {
			return s.forbidden(c, "tx not allowed to execute", err)
		}
	} else {
		_, err = ngn.Evaluate(recipe, firstKeysignMessage.Chain, txBytesEvaluate)
		if err != nil {
			return s.forbidden(c, "tx not allowed to execute", err)
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
		errMsg := "failed to create tx for tracking"
		return s.internal(c, errMsg, err)
	}
	req.Messages[0].TxIndexerID = txToTrack.ID.String()

	err = s.txIndexerService.SetStatus(c.Request().Context(), txToTrack.ID, storage.TxVerified)
	if err != nil {
		errMsg := fmt.Sprintf("tx_id=%s, failed to set transaction status to verified", txToTrack.ID)
		return s.internal(c, errMsg, err)
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
		errMsg := "fail to marshal to json"
		return s.badRequest(c, errMsg, err)
	}

	ti, err := s.asynqClient.EnqueueContext(c.Request().Context(),
		asynq.NewTask(tasks.TypeKeySignDKLS, buf),
		asynq.MaxRetry(0),
		asynq.Timeout(2*time.Minute),
		asynq.Retention(5*time.Minute),
		asynq.Queue(tasks.QUEUE_NAME))

	if err != nil {
		errMsg := "fail to enqueue keysign task"
		return s.internal(c, errMsg, err)
	}

	taskIDsResponse := map[string][]string{
		"task_ids": {ti.ID},
	}
	return c.JSON(http.StatusOK, NewSuccessResponse(http.StatusOK, taskIDsResponse))
}

func (s *Server) GetPlugins(c echo.Context) error {
	skip, take, err := conv.PageParamsFromCtx(c, 0, 20)
	if err != nil {
		return s.badRequest(c, msgInvalidPagination, err)
	}
	sort := c.QueryParam("sort")
	filters := types.PluginFilters{
		Term:       common.GetQueryParam(c, "term"),
		TagID:      common.GetUUIDParam(c, "tag_id"),
		CategoryID: common.GetQueryParam(c, "category_id"),
	}

	plugins, err := s.db.FindPlugins(c.Request().Context(), filters, int(take), int(skip), sort)
	if err != nil {
		return s.internal(c, msgGetPluginsFailed, err)
	}

	return c.JSON(http.StatusOK, NewSuccessResponse(http.StatusOK, plugins))
}

func (s *Server) GetPlugin(c echo.Context) error {
	pluginID := c.Param("pluginId")
	if pluginID == "" {
		return s.badRequest(c, msgRequiredPluginID, nil)
	}

	plugin, err := s.pluginService.GetPluginWithRating(c.Request().Context(), pluginID)
	if err != nil {
		return s.internal(c, msgGetPluginFailed, err)
	}

	return c.JSON(http.StatusOK, NewSuccessResponse(http.StatusOK, plugin))
}

func (s *Server) GetInstalledPlugins(c echo.Context) error {
	publicKey, ok := c.Get("vault_public_key").(string)
	if !ok || publicKey == "" {
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgVaultPublicKeyGetFailed))
	}

	// TODO(performance):
	// This implementation checks plugins one-by-one using storage.Exist().
	// This is NOT scalable because our current filename format:
	//   {pluginId}-{publicKey}.vult
	// prevents prefix-based listing.
	//
	// We SHOULD change the storage key format to:
	//   vaults/{publicKey}/{pluginId}.vult
	// and use List(prefix) instead of N calls to Exist().
	// This will drastically improve performance and reduce storage load.

	pluginList, err := s.db.FindPlugins(c.Request().Context(), types.PluginFilters{}, 1000, 0, "")
	if err != nil {
		return s.internal(c, msgGetPluginsFailed, err)
	}

	var installed types.PluginsPaginatedList
	installed.Plugins = make([]types.Plugin, 0, len(pluginList.Plugins))

	for _, plugin := range pluginList.Plugins {
		fileName := common.GetVaultBackupFilename(publicKey, string(plugin.ID))

		exist, err := s.vaultStorage.Exist(fileName)
		if err != nil {
			s.logger.WithError(err).Errorf("failed to check vault existence for plugin %s", plugin.ID)
			continue
		}
		if exist {
			installed.Plugins = append(installed.Plugins, plugin)
			installed.TotalCount++
		}
	}

	return c.JSON(http.StatusOK, NewSuccessResponse(http.StatusOK, installed))
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
			ID:   string(types.PluginCategoryApp),
			Name: types.PluginCategoryApp.String(),
		},
	}
	return c.JSON(http.StatusOK, NewSuccessResponse(http.StatusOK, resp))
}

func (s *Server) GetTags(c echo.Context) error {
	tags, err := s.db.FindTags(c.Request().Context())
	if err != nil {
		return s.internal(c, msgGetTagsFailed, err)
	}
	return c.JSON(http.StatusOK, NewSuccessResponse(http.StatusOK, tags))
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
	if !ok || publicKey == "" {
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
		return s.internal(c, msgGetTxsByPolicyIDFailed, err)
	}

	return c.JSON(http.StatusOK, NewSuccessResponse(http.StatusOK, types.TransactionHistoryPaginatedList{
		History:    txs,
		TotalCount: totalCount,
	}))
}

func (s *Server) CreateReview(c echo.Context) error {
	var review types.ReviewCreateDto
	if err := c.Bind(&review); err != nil {
		return s.badRequest(c, msgRequestParseFailed, err)
	}

	if err := c.Validate(&review); err != nil {
		return s.badRequest(c, msgInvalidReview, err)
	}

	// If allowing HTML, sanitize with bluemonday:
	p := bluemonday.UGCPolicy()
	review.Comment = p.Sanitize(review.Comment)

	pluginID := c.Param("pluginId")
	if pluginID == "" {
		return s.badRequest(c, msgRequiredPluginID, nil)
	}

	created, err := s.pluginService.CreatePluginReviewWithRating(c.Request().Context(), review, pluginID)
	if err != nil {
		return s.internal(c, msgCreateReviewFailed, err)
	}

	return c.JSON(http.StatusOK, NewSuccessResponse(http.StatusOK, created))
}

func (s *Server) GetReviews(c echo.Context) error {
	pluginId := c.Param("pluginId")
	if pluginId == "" {
		return s.badRequest(c, msgRequiredPluginID, nil)
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
		return s.badRequest(c, msgInvalidSort, nil)
	}

	reviews, err := s.db.FindReviews(c.Request().Context(), pluginId, skip, take, sort)
	if err != nil {
		return s.internal(c, msgGetReviewsFailed, err)
	}

	return c.JSON(http.StatusOK, NewSuccessResponse(http.StatusOK, reviews))
}

func (s *Server) GetPluginAvgRating(c echo.Context) error {
	pluginID := c.Param("pluginId")
	if pluginID == "" {
		return s.badRequest(c, msgRequiredPluginID, nil)
	}

	avgRating, err := s.db.FindAvgRatingByPluginID(c.Request().Context(), pluginID)
	if err != nil {
		return s.internal(c, msgGetAvgRatingFailed, err)
	}

	return c.JSON(http.StatusOK, NewSuccessResponse(http.StatusOK, avgRating))
}

func (s *Server) GetPluginRecipeSpecification(c echo.Context) error {
	pluginID := c.Param("pluginId")
	if pluginID == "" {
		return s.badRequest(c, msgRequiredPluginID, nil)
	}
	recipeSpec, err := s.pluginService.GetPluginRecipeSpecification(c.Request().Context(), pluginID)
	if err != nil {
		return s.internal(c, msgGetRecipeSpecFailed, err)
	}

	return c.JSON(http.StatusOK, NewSuccessResponse(http.StatusOK, recipeSpec))
}

func (s *Server) GetPluginRecipeSpecificationSuggest(c echo.Context) error {
	pluginID := c.Param("pluginId")
	if pluginID == "" {
		return s.badRequest(c, msgRequiredPluginID, nil)
	}

	type reqBody struct {
		Configuration map[string]any `json:"configuration"`
	}
	var req reqBody
	err := c.Bind(&req)
	if err != nil {
		return s.badRequest(c, msgRequestParseFailed, err)
	}

	recipeSpec, err := s.pluginService.GetPluginRecipeSpecificationSuggest(
		c.Request().Context(),
		pluginID,
		req.Configuration,
	)
	if err != nil {
		return s.internal(c, msgGetRecipeSuggestFailed, err)
	}

	b, err := protojson.Marshal(recipeSpec)
	if err != nil {
		return s.internal(c, msgProtoMarshalFailed, err)
	}

	var res map[string]any
	err = json.Unmarshal(b, &res)
	if err != nil {
		return s.internal(c, msgJSONMarshalFailed, err)
	}

	return c.JSON(http.StatusOK, NewSuccessResponse(http.StatusOK, res))
}

func (s *Server) GetPluginRecipeFunctions(c echo.Context) error {
	pluginID := c.Param("pluginId")
	if pluginID == "" {
		return s.badRequest(c, msgRequiredPluginID, nil)
	}
	recipeFuncs, err := s.pluginService.GetPluginRecipeFunctions(c.Request().Context(), pluginID)
	if err != nil {
		return s.internal(c, msgGetRecipeFunctionsFailed, err)
	}

	return c.JSON(http.StatusOK, NewSuccessResponse(http.StatusOK, recipeFuncs))
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

	return c.JSON(http.StatusOK, NewSuccessResponse(http.StatusOK, count))
}

func (s *Server) badRequest(c echo.Context, msg string, err error) error {
	if err != nil {
		s.logger.WithError(err).Error(msg)
	} else {
		s.logger.Warn(msg)
	}
	return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage(msg))
}

func (s *Server) internal(c echo.Context, msg string, err error) error {
	if err != nil {
		s.logger.WithError(err).Error(msg)
	} else {
		s.logger.Error(msg)
	}
	return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgInternalError))
}

func (s *Server) forbidden(c echo.Context, msg string, err error) error {
	if err != nil {
		s.logger.WithError(err).Error(msg)
	} else {
		s.logger.Warn(msg)
	}
	return c.JSON(http.StatusForbidden, NewErrorResponseWithMessage(msg))
}
