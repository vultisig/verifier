package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"
	"github.com/microcosm-cc/bluemonday"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/vultisig/recipes/engine"
	sdk "github.com/vultisig/recipes/sdk"
	sdkbch "github.com/vultisig/recipes/sdk/bch"
	sdkbtc "github.com/vultisig/recipes/sdk/btc"
	sdkcosmos "github.com/vultisig/recipes/sdk/cosmos"
	sdkevm "github.com/vultisig/recipes/sdk/evm"
	sdksolana "github.com/vultisig/recipes/sdk/solana"
	sdktron "github.com/vultisig/recipes/sdk/tron"
	sdkxrpl "github.com/vultisig/recipes/sdk/xrpl"
	sdkzcash "github.com/vultisig/recipes/sdk/zcash"
	rtypes "github.com/vultisig/recipes/types"
	"github.com/vultisig/verifier/internal/conv"
	"github.com/vultisig/verifier/internal/safety"
	"github.com/vultisig/verifier/internal/service"
	"github.com/vultisig/verifier/internal/types"
	"github.com/vultisig/verifier/plugin/scheduler"
	"github.com/vultisig/verifier/plugin/tasks"
	"github.com/vultisig/verifier/plugin/tx_indexer/pkg/storage"
	vtypes "github.com/vultisig/verifier/types"
	"github.com/vultisig/vultisig-go/common"
)

func (s *Server) SignPluginMessages(c echo.Context) error {
	s.logger.Debug("PLUGIN SERVER: SIGN MESSAGES")

	var req vtypes.PluginKeysignRequest
	if err := c.Bind(&req); err != nil {
		errMsg := "fail to parse request"
		return s.badRequest(c, errMsg, err)
	}

	// Verify authenticated plugin ID matches the requested plugin ID
	// This prevents a malicious plugin from impersonating another plugin
	authenticatedPluginID, ok := c.Get("plugin_id").(vtypes.PluginID)
	if !ok {
		return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage(msgRequiredPluginID))
	}
	if authenticatedPluginID.String() != req.PluginID {
		s.logger.Warnf("Plugin ID mismatch: authenticated=%s, requested=%s", authenticatedPluginID, req.PluginID)
		return c.JSON(http.StatusForbidden, NewErrorResponseWithMessage(msgPluginIDMismatch))
	}

	if err := s.safetyMgm.EnforceKeysign(c.Request().Context(), req.PluginID); err != nil {
		if safety.IsDisabledError(err) {
			s.logger.WithError(err).WithField("plugin_id", req.PluginID).Warn("SignPluginMessages: Plugin is paused")
			return c.JSON(http.StatusLocked, NewErrorResponseWithMessage(msgPluginPaused))
		}
		s.logger.WithError(err).WithField("plugin_id", req.PluginID).Error("SignPluginMessages: EnforceKeysign failed")
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgRequestProcessFailed))
	}

	// Get policy from database
	if req.PluginID == vtypes.PluginVultisigFees_feee.String() {
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
			filePathName := common.GetVaultBackupFilename(req.PublicKey, vtypes.PluginVultisigFees_feee.String())
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

		// Check if policy is inactive
		if !policy.Active {
			return c.JSON(http.StatusForbidden, NewErrorResponseWithMessage(msgPolicyEnded))
		}

		// Validate policy matches plugin
		if policy.PluginID != vtypes.PluginID(req.PluginID) {
			return fmt.Errorf("policy plugin ID mismatch")
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

		// Perform signing
		signErr := s.validateAndSign(c, &req, recipe, policy.ID)

		// After signing, check if policy should be deactivated
		s.checkAndDeactivatePolicy(c.Request().Context(), policy, recipe)

		return signErr
	}
}

// checkAndDeactivatePolicy checks if a policy has no more executions and deactivates it.
// For policies with rateLimitWindow, deactivation is deferred via a background task.
func (s *Server) checkAndDeactivatePolicy(ctx context.Context, policy *vtypes.PluginPolicy, recipe *rtypes.Policy) {
	interval := scheduler.NewDefaultInterval()
	next, err := interval.FromNowWhenNext(*policy)
	if err != nil {
		// Plugin uses custom recipe format - skip deactivation check
		return
	}

	if !next.IsZero() {
		return
	}

	// Policy has no more scheduled executions - check if deferred deactivation needed
	if recipe.RateLimitWindow != nil && recipe.GetRateLimitWindow() > 0 {
		delay := time.Duration(recipe.GetRateLimitWindow()) * time.Second
		_, err = s.asynqClient.EnqueueContext(
			ctx,
			asynq.NewTask(tasks.TypePolicyDeactivate, []byte(policy.ID.String())),
			asynq.ProcessIn(delay),
			asynq.MaxRetry(3),
			asynq.Queue(tasks.QUEUE_NAME),
			asynq.TaskID(fmt.Sprintf("deactivate:%s", policy.ID)),
		)
		if errors.Is(err, asynq.ErrTaskIDConflict) {
			// Deactivation already scheduled - this is expected for multi-tx operations
			return
		}
		if err != nil {
			s.logger.WithError(err).Warnf("policy_id=%s: failed to schedule deactivation", policy.ID)
		}
		return
	}

	// No rateLimitWindow - deactivate immediately
	s.logger.Infof("policy_id=%s: no rateLimitWindow, deactivating immediately", policy.ID)
	policy.Deactivate(vtypes.DeactivationReasonCompleted)
	_, err = s.policyService.UpdatePolicy(ctx, *policy)
	if err != nil {
		s.logger.WithError(err).Errorf("policy_id=%s: failed to deactivate", policy.ID)
	} else {
		s.logger.Infof("policy_id=%s: deactivated immediately", policy.ID)
	}
}

func (s *Server) validateAndSign(c echo.Context, req *vtypes.PluginKeysignRequest, recipe *rtypes.Policy, policyID uuid.UUID) error {
	if len(req.Messages) == 0 {
		return s.badRequest(c, msgNoMessagesToSign, nil)
	}

	firstKeysignMessage := req.Messages[0]

	ngn, err := engine.NewEngine()
	if err != nil {
		return s.internal(c, "failed to create engine", err)
	}

	// Extract transaction bytes using chain-specific handler
	transactionPreview := req.Transaction
	if len(transactionPreview) > 100 {
		transactionPreview = transactionPreview[:100]
	}
	fmt.Printf("[VERIFIER DEBUG] chain=%s Transaction length=%d, first 100 chars: %s\n",
		firstKeysignMessage.Chain.String(),
		len(req.Transaction),
		transactionPreview)
	txBytesEvaluate, err := ngn.ExtractTxBytes(firstKeysignMessage.Chain, req.Transaction)
	if err != nil {
		return s.badRequest(c, "failed to extract transaction bytes", err)
	}

	// SECURITY: Derive signing hashes from txBytes, ignore plugin-provided hashes.
	// This prevents a malicious plugin from sending txBytes for validation but a different hash for signing.
	derivedHashes, err := deriveSigningHashes(firstKeysignMessage.Chain, txBytesEvaluate, req.Transaction, req.SignBytes)
	if err != nil {
		return s.badRequest(c, "failed to derive signing hash", err)
	}

	// Verify message count matches derived hash count (important for Bitcoin multi-input)
	if len(derivedHashes) != len(req.Messages) {
		return s.badRequest(c, fmt.Sprintf("expected %d messages for %d derived hashes, got %d messages",
			len(derivedHashes), len(derivedHashes), len(req.Messages)), nil)
	}

	// Replace plugin-provided Message/Hash with verifier-derived values
	for i := range req.Messages {
		req.Messages[i].Message = base64.StdEncoding.EncodeToString(derivedHashes[i].Message)
		req.Messages[i].Hash = base64.StdEncoding.EncodeToString(derivedHashes[i].Hash)
		req.Messages[i].HashFunction = vtypes.HashFunction_SHA256
	}

	var matchedRule *rtypes.Rule
	//TODO: fee plugin priority for testing purposes
	if req.PluginID == vtypes.PluginVultisigFees_feee.String() {
		matchedRule, err = ngn.Evaluate(types.FeeDefaultPolicy, firstKeysignMessage.Chain, txBytesEvaluate)
		if err != nil {
			return s.forbidden(c, msgTxNotAllowed, err)
		}
	} else {
		matchedRule, err = ngn.Evaluate(recipe, firstKeysignMessage.Chain, txBytesEvaluate)
		if err != nil {
			return s.forbidden(c, msgTxNotAllowed, err)
		}
	}

	// Extract transaction details from matched rule's parameter constraints
	amount := extractAmountFromRule(matchedRule)
	tokenID := extractTokenIDFromRule(matchedRule)
	toAddress := extractToAddressFromRule(matchedRule)

	txToTrack, err := s.txIndexerService.CreateTx(c.Request().Context(), storage.CreateTxDto{
		PluginID:      vtypes.PluginID(req.PluginID),
		ChainID:       firstKeysignMessage.Chain,
		PolicyID:      policyID,
		TokenID:       tokenID,
		FromPublicKey: req.PublicKey,
		ToPublicKey:   toAddress,
		ProposedTxHex: req.Transaction,
		Amount:        amount,
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

	// Build title map from unique plugin IDs
	titleMap, err := s.buildPluginTitleMap(c.Request().Context(), txs)
	if err != nil {
		s.logger.WithError(err).Error("s.buildPluginTitleMap")
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgGetPluginFailed))
	}

	return c.JSON(http.StatusOK, NewSuccessResponse(http.StatusOK, types.TransactionHistoryPaginatedList{
		History:    types.FromStorageTxs(txs, titleMap),
		TotalCount: totalCount,
	}))
}

func (s *Server) GetPluginTransactionHistory(c echo.Context) error {
	pluginID := c.QueryParam("pluginId") // Optional query parameter

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

	var txs []storage.Tx
	var totalCount uint32

	if pluginID != "" {
		// Filter by specific plugin
		txs, totalCount, err = s.txIndexerService.GetByPluginIDAndPublicKey(
			c.Request().Context(),
			vtypes.PluginID(pluginID),
			publicKey,
			skip,
			take,
		)
		if err != nil {
			s.logger.WithError(err).Errorf("s.txIndexerService.GetByPluginIDAndPublicKey: %s", pluginID)
			return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgGetTxsByPluginIDFailed))
		}
	} else {
		// Get all transactions for the user across all plugins
		txs, totalCount, err = s.txIndexerService.GetByPublicKey(
			c.Request().Context(),
			publicKey,
			skip,
			take,
		)
		if err != nil {
			s.logger.WithError(err).Errorf("s.txIndexerService.GetByPublicKey: %s", publicKey)
			return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgGetTxsByPluginIDFailed))
		}
	}

	// Build title map from unique plugin IDs
	titleMap, err := s.buildPluginTitleMap(c.Request().Context(), txs)
	if err != nil {
		s.logger.WithError(err).Error("s.buildPluginTitleMap")
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgGetPluginFailed))
	}

	return c.JSON(http.StatusOK, NewSuccessResponse(http.StatusOK, types.TransactionHistoryPaginatedList{
		History:    types.FromStorageTxs(txs, titleMap),
		TotalCount: totalCount,
	}))
}

func (s *Server) buildPluginTitleMap(ctx context.Context, txs []storage.Tx) (map[string]string, error) {
	// Get unique plugin IDs
	pluginIDSet := make(map[string]struct{})
	for _, tx := range txs {
		pluginIDSet[string(tx.PluginID)] = struct{}{}
	}

	if len(pluginIDSet) == 0 {
		return make(map[string]string), nil
	}

	pluginIDs := make([]string, 0, len(pluginIDSet))
	for id := range pluginIDSet {
		pluginIDs = append(pluginIDs, id)
	}

	return s.pluginService.GetPluginTitlesByIDs(ctx, pluginIDs)
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

	count, err := s.policyService.GetPluginInstallationsCount(c.Request().Context(), vtypes.PluginID(pluginID))
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

// extractAmountFromRule extracts the amount from a matched rule's parameter constraints.
// For send transactions, it looks for "amount" parameter.
// For swap transactions, it looks for "from_amount" parameter.
func extractAmountFromRule(rule *rtypes.Rule) string {
	if rule == nil {
		return ""
	}

	for _, pc := range rule.GetParameterConstraints() {
		paramName := pc.GetParameterName()
		// Check for send amount or swap from_amount
		if paramName == "amount" || paramName == "from_amount" {
			constraint := pc.GetConstraint()
			if constraint != nil {
				// Try to get fixed value first (most common for recurring send/swap)
				if fixedVal := constraint.GetFixedValue(); fixedVal != "" {
					return fixedVal
				}
				// For other constraint types, we can't determine the exact amount
			}
		}
	}

	return ""
}

// extractTokenIDFromRule extracts the token address from a matched rule.
// After metarule processing:
// - For ERC20 sends: Target.Address contains the token contract address
// - For 1inch swaps: "desc.srcToken" parameter contains the source token
// Returns empty string for native token transfers.
func extractTokenIDFromRule(rule *rtypes.Rule) string {
	if rule == nil {
		return ""
	}

	// Case 1: ERC20 transfer - the Target.Address is the token contract
	resource := rule.GetResource()
	if strings.Contains(resource, ".erc20.transfer") {
		if target := rule.GetTarget(); target != nil {
			if addr := target.GetAddress(); addr != "" {
				return addr
			}
		}
	}

	// Case 2: Check parameter constraints for 1inch swap (srcToken)
	for _, pc := range rule.GetParameterConstraints() {
		paramName := pc.GetParameterName()
		if paramName == "srcToken" {
			constraint := pc.GetConstraint()
			if constraint != nil {
				if fixedVal := constraint.GetFixedValue(); fixedVal != "" {
					return fixedVal
				}
			}
		}
	}

	return ""
}

// extractToAddressFromRule extracts the recipient address from a matched rule.
// After metarule processing, checks various parameter names:
// - EVM ERC20: "recipient"
// - EVM native: Target.Address
// - 1inch swap: "desc.dstReceiver"
// - Solana native: "account_to"
// - Solana SPL: "account_destination"
// - Bitcoin/UTXO: "output_address_0"
// - XRP/THORChain: "recipient"
func extractToAddressFromRule(rule *rtypes.Rule) string {
	if rule == nil {
		return ""
	}

	// For EVM native transfers, the recipient is in Target.Address
	resource := rule.GetResource()
	if strings.Contains(resource, ".eth.transfer") ||
		strings.Contains(resource, ".bnb.transfer") ||
		strings.Contains(resource, ".matic.transfer") ||
		strings.Contains(resource, ".avax.transfer") {
		if target := rule.GetTarget(); target != nil {
			if addr := target.GetAddress(); addr != "" {
				return addr
			}
		}
	}

	// Check parameter constraints for various recipient parameter names
	recipientParams := []string{
		"recipient",           // EVM ERC20, XRP, THORChain
		"dstReceiver",         // 1inch swap
		"account_to",          // Solana native
		"account_destination", // Solana SPL
		"output_address_0",    // Bitcoin/UTXO
	}

	for _, pc := range rule.GetParameterConstraints() {
		paramName := pc.GetParameterName()
		for _, recipientParam := range recipientParams {
			if paramName == recipientParam {
				constraint := pc.GetConstraint()
				if constraint != nil {
					if fixedVal := constraint.GetFixedValue(); fixedVal != "" {
						return fixedVal
					}
				}
			}
		}
	}

	return ""
}

func (s *Server) ReportPlugin(c echo.Context) error {
	pluginID := c.Param("pluginId")
	if pluginID == "" {
		return s.badRequest(c, msgRequiredPluginID, nil)
	}

	publicKey, ok := c.Get("vault_public_key").(string)
	if !ok || publicKey == "" {
		return c.JSON(http.StatusInternalServerError, NewErrorResponseWithMessage(msgVaultPublicKeyGetFailed))
	}

	var req types.ReportCreateRequest
	err := c.Bind(&req)
	if err != nil {
		return s.badRequest(c, msgRequestParseFailed, err)
	}

	err = c.Validate(&req)
	if err != nil {
		return s.badRequest(c, msgReasonRequired, err)
	}

	p := bluemonday.StrictPolicy()
	reason := p.Sanitize(req.Reason)

	result, err := s.reportService.SubmitReport(c.Request().Context(), vtypes.PluginID(pluginID), publicKey, reason)
	if err != nil {
		if errors.Is(err, service.ErrNotEligible) {
			return c.JSON(http.StatusBadRequest, NewErrorResponseWithMessage(msgReportNotEligible))
		}
		if errors.Is(err, service.ErrCooldownActive) {
			return c.JSON(http.StatusTooManyRequests, NewErrorResponseWithMessage(err.Error()))
		}
		return s.internal(c, "failed to submit report", err)
	}

	return c.JSON(http.StatusOK, NewSuccessResponse(http.StatusOK, result))
}

// deriveSigningHashes derives signing hashes from transaction bytes based on chain type.
// This is the core security function that ensures the verifier derives hashes independently,
// preventing malicious plugins from substituting different hashes for signing.
func deriveSigningHashes(chain common.Chain, txBytes []byte, originalTx string, signBytesBase64 string) ([]sdk.DerivedHash, error) {
	opts := sdk.DeriveOptions{}

	switch {
	case chain.IsEvm():
		evmID, err := chain.EvmID()
		if err != nil {
			return nil, fmt.Errorf("failed to get EVM chain ID: %w", err)
		}
		evmSDK := sdkevm.NewSDK(evmID, nil, nil)
		return evmSDK.DeriveSigningHashes(txBytes, opts)

	case chain == common.Solana:
		solanaSDK := sdksolana.NewSDK(nil)
		return solanaSDK.DeriveSigningHashes(txBytes, opts)

	case chain == common.BitcoinCash:
		bchSDK := sdkbch.NewSDK(nil)
		psbtBytes, err := base64.StdEncoding.DecodeString(originalTx)
		if err != nil {
			return nil, fmt.Errorf("failed to decode PSBT base64: %w", err)
		}
		return bchSDK.DeriveSigningHashes(psbtBytes, opts)

	case chain == common.Bitcoin, chain == common.Litecoin, chain == common.Dogecoin, chain == common.Dash:
		btcSDK := sdkbtc.NewSDK(nil)
		// UTXO chains need the full PSBT (not extracted tx bytes) to calculate signature hashes,
		// because they require the WitnessUtxo/NonWitnessUtxo info from the PSBT inputs.
		// The originalTx is base64-encoded, so we need to decode it first.
		psbtBytes, err := base64.StdEncoding.DecodeString(originalTx)
		if err != nil {
			return nil, fmt.Errorf("failed to decode PSBT base64: %w", err)
		}
		return btcSDK.DeriveSigningHashes(psbtBytes, opts)

	case chain == common.XRP:
		xrpSDK := sdkxrpl.NewSDK(nil)
		return xrpSDK.DeriveSigningHashes(txBytes, opts)

	case chain == common.THORChain, chain == common.MayaChain:
		// Cosmos-based chains require signBytes to be provided
		if signBytesBase64 == "" {
			return nil, fmt.Errorf("sign_bytes required for Cosmos-based chain %s", chain.String())
		}
		signBytes, err := base64.StdEncoding.DecodeString(signBytesBase64)
		if err != nil {
			return nil, fmt.Errorf("failed to decode sign_bytes: %w", err)
		}
		opts.SignBytes = signBytes
		cosmosSDK := sdkcosmos.NewSDK(nil)
		return cosmosSDK.DeriveSigningHashes(txBytes, opts)

	case chain == common.Tron:
		tronSDK := sdktron.NewSDK(nil)
		return tronSDK.DeriveSigningHashes(txBytes, opts)

	case chain == common.Zcash:
		zcashSDK := sdkzcash.NewSDK(nil)
		// Zcash uses the original transaction which contains embedded metadata (sighashes + pubkey)
		zcashBytes, err := base64.StdEncoding.DecodeString(originalTx)
		if err != nil {
			return nil, fmt.Errorf("failed to decode Zcash transaction base64: %w", err)
		}
		return zcashSDK.DeriveSigningHashes(zcashBytes, opts)

	default:
		return nil, fmt.Errorf("unsupported chain for hash derivation: %s", chain.String())
	}
}
