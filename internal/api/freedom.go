package api

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
	"github.com/labstack/echo/v4"
	"github.com/vultisig/recipes/engine"
	"github.com/vultisig/verifier/plugin/tasks"
	"github.com/vultisig/verifier/plugin/tx_indexer/pkg/storage"
	vtypes "github.com/vultisig/verifier/types"
	"github.com/vultisig/vultisig-go/common"
)

// FreedomSign is the co-signer handler for the freedom signing surface.
//
// Design:
//   - Authenticated via JWT (VaultAuthMiddleware) — the signer's public key is
//     extracted from the token, NOT from the request body.
//   - The freedom policy (FreedomPolicy) is evaluated against the proposed tx.
//     When cfg.Freedom.Enforce is true (tests / opt-in prod), a policy miss → 403.
//     When false (default shadow mode), the policy miss is logged but signing
//     proceeds — allows ramping the gate without blocking users.
//   - PluginID is hardcoded to FreedomPluginID; the client cannot override it.
//   - Hash derivation mirrors validateAndSign: the verifier derives hashes from
//     txBytes independently, preventing the "bytes X, hash Y" substitution attack.
func (s *Server) FreedomSign(c echo.Context) error {
	// Public key from JWT — authenticated, not client-controlled.
	publicKey, ok := c.Get("vault_public_key").(string)
	if !ok || publicKey == "" {
		return c.JSON(http.StatusUnauthorized, NewErrorResponseWithMessage(msgUnauthorized))
	}

	var req vtypes.PluginKeysignRequest
	if err := c.Bind(&req); err != nil {
		return s.badRequest(c, msgRequestParseFailed, err)
	}

	// Override with server-derived values — never trust the client for these.
	req.PublicKey = publicKey
	req.PluginID = FreedomPluginID

	if len(req.Messages) == 0 {
		return s.badRequest(c, msgNoMessagesToSign, nil)
	}
	firstMsg := req.Messages[0]

	// Reject chains the freedom policy doesn't cover — fail-closed in both shadow
	// and enforce mode. Without this guard, an unrecognised chain (Solana, BTC,
	// THORChain, …) would reach policy evaluation with zero matching rules:
	//   - shadow mode: "no matching rule" is treated as a pass → anything gets signed
	//   - enforce mode: same miss → policy error logged but no explicit rejection here
	// Fail-closed here so the gate is meaningful regardless of mode.
	supportedChains := supportedFreedomChains()
	chainSupported := false
	for _, sc := range supportedChains {
		if firstMsg.Chain == sc {
			chainSupported = true
			break
		}
	}
	if !chainSupported {
		return s.badRequest(c, fmt.Sprintf("chain %q is not supported by the freedom policy", firstMsg.Chain), nil)
	}

	ngn, err := engine.NewEngine()
	if err != nil {
		return s.internal(c, "failed to create engine", err)
	}

	// Extract tx bytes using the chain-specific handler.
	txBytesEvaluate, err := ngn.ExtractTxBytes(firstMsg.Chain, req.Transaction)
	if err != nil {
		return s.badRequest(c, "failed to extract transaction bytes", err)
	}

	// Derive signing hashes from the verified tx bytes — ignores any client-provided hashes.
	derivedHashes, err := deriveSigningHashes(firstMsg.Chain, txBytesEvaluate, req.Transaction, req.SignBytes)
	if err != nil {
		return s.badRequest(c, "failed to derive signing hash", err)
	}

	if len(derivedHashes) != len(req.Messages) {
		return s.badRequest(c, fmt.Sprintf("expected %d messages for %d derived hashes, got %d messages",
			len(derivedHashes), len(derivedHashes), len(req.Messages)), nil)
	}

	// Replace client hashes with verifier-derived values.
	for i := range req.Messages {
		req.Messages[i].Message = base64.StdEncoding.EncodeToString(derivedHashes[i].Message)
		req.Messages[i].Hash = base64.StdEncoding.EncodeToString(derivedHashes[i].Hash)
		req.Messages[i].HashFunction = vtypes.HashFunction_SHA256
	}

	// Evaluate the freedom policy.
	_, policyErr := ngn.Evaluate(FreedomPolicy, firstMsg.Chain, txBytesEvaluate)
	if policyErr != nil {
		if s.cfg.Freedom.Enforce {
			return s.forbidden(c, msgTxNotAllowed, policyErr)
		}
		// Shadow mode: log but proceed.
		s.logger.WithError(policyErr).Warn("freedom policy: tx not allowed (shadow mode, proceeding)")
	}

	// Verify the freedom vault has been onboarded before enqueuing a keysign.
	// Mirrors the existence gate in validateAndSign (plugin.go). Without this,
	// a valid JWT for a not-yet-onboarded vault would queue a keysign that the
	// worker would fail to execute, burning a signing slot and polluting the
	// tx indexer.
	vaultFile := common.GetVaultBackupFilename(req.PublicKey, FreedomPluginID)
	vaultExists, err := s.vaultStorage.Exist(vaultFile)
	if err != nil {
		return s.internal(c, "failed to check freedom vault existence", err)
	}
	if !vaultExists {
		return c.JSON(http.StatusNotFound, NewErrorResponseWithMessage("freedom vault not found — vault must be onboarded before signing"))
	}

	// Track the transaction.
	txToTrack, err := s.txIndexerService.CreateTx(c.Request().Context(), storage.CreateTxDto{
		PluginID:      vtypes.PluginID(req.PluginID),
		ChainID:       firstMsg.Chain,
		PolicyID:      uuid.Nil,
		TokenID:       "",
		FromPublicKey: req.PublicKey,
		ToPublicKey:   "",
		ProposedTxHex: req.Transaction,
		Amount:        "",
	})
	if err != nil {
		return s.internal(c, "failed to create tx for tracking", err)
	}
	req.Messages[0].TxIndexerID = txToTrack.ID.String()

	if err := s.txIndexerService.SetStatus(c.Request().Context(), txToTrack.ID, storage.TxVerified); err != nil {
		return s.internal(c, fmt.Sprintf("tx_id=%s, failed to set transaction status to verified", txToTrack.ID), err)
	}

	// Deduplicate by session ID (idempotent re-submissions return 200).
	result, err := s.redis.Get(c.Request().Context(), req.SessionID)
	if err == nil && result != "" {
		return c.NoContent(http.StatusOK)
	}

	if err := s.redis.Set(c.Request().Context(), req.SessionID, req.SessionID, 30*time.Minute); err != nil {
		s.logger.WithError(err).Error("fail to set session")
	}

	buf, err := json.Marshal(req)
	if err != nil {
		return s.badRequest(c, "fail to marshal to json", err)
	}

	ti, err := s.asynqClient.EnqueueContext(c.Request().Context(),
		asynq.NewTask(tasks.TypeKeySignDKLS, buf),
		asynq.MaxRetry(0),
		asynq.Timeout(2*time.Minute),
		asynq.Retention(5*time.Minute),
		asynq.Queue(tasks.QUEUE_NAME))
	if err != nil {
		return s.internal(c, "fail to enqueue keysign task", err)
	}

	return c.JSON(http.StatusOK, NewSuccessResponse(http.StatusOK, map[string][]string{
		"task_ids": {ti.ID},
	}))
}

// supportedFreedomChains returns the set of chains the freedom policy covers.
// Only chains with matching rules in FreedomPolicy are included; requests for
// other chains will fail policy evaluation.
func supportedFreedomChains() []common.Chain {
	return []common.Chain{common.Ethereum}
}
