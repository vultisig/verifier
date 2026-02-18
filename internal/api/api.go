package api

import "time"

type APIResponse[T any] struct {
	Data      T             `json:"data,omitempty"`
	Error     ErrorResponse `json:"error"`
	Status    int           `json:"status,omitempty"`
	Timestamp string        `json:"timestamp"`
	Version   string        `json:"version"`
}

type ErrorResponse struct {
	Message          string `json:"message"`
	DetailedResponse string `json:"details,omitempty"`
}

// Error messages
const (
	msgMissingAuthHeader   = "missing authorization header"
	msgInvalidAuthHeader   = "invalid authorization header format"
	msgUnauthorized        = "unauthorized"
	msgInternalError       = "an internal error occurred"
	msgAccessDenied        = "access denied: token not authorized for this vault"
	msgAccessDeniedBilling = "access denied: install billing app first"

	// Token
	msgMissingTokenID              = "missing tokenId"
	msgTokenGenerateFailed         = "failed to generate auth token"
	msgInvalidOrExpiredToken       = "invalid or expired token"
	msgRevokeAllTokensFailed       = "failed to revoke all tokens"
	msgGetActiveTokensFailed       = "failed to get active tokens"
	msgUnauthorizedTokenRevocation = "unauthorized token revocation"
	msgTokenRevokeFailed           = "failed to revoke token"
	msgTokenNotFound               = "token not found"
	msgTokenGetFailed              = "failed to get token"

	// API key
	msgDisabledAPIKey = "API key is disabled"
	msgExpiredAPIKey  = "API key has expired"
	msgAPIKeyNotFound = "API key not found"

	// Plugin
	msgRequiredPluginID              = "pluginId is required"
	msgPluginInstallationCountFailed = "failed to get plugin installation count"
	msgGetPluginFailed               = "failed to get plugin"
	msgGetPluginsFailed              = "failed to get plugins"
	msgGetAvgRatingFailed            = "failed to get average rating"
	msgPluginServerUnavailable       = "plugin server is currently unavailable"
	msgPluginPaused                  = "plugin is currently paused"
	msgPluginIDMismatch              = "plugin ID mismatch: not authorized to sign for this plugin"

	// Policy reactivation
	msgReactivateMissingReason = "cannot reactivate policy without deactivation_reason"
	msgReactivateExpiredPolicy = "cannot reactivate expired/completed policy"
	msgReactivateInvalidReason = "cannot reactivate policy with this deactivation reason"

	// Plugin Report
	msgReportNotEligible      = "not eligible to report: no installation found"
	msgReportCooldownActive   = "cooldown active: please wait before reporting again"
	msgReportValidationFailed = "validation failed"
	msgReportReasonRequired   = "reason is required"
	msgReportReasonTooLong    = "reason must be at most 200 characters"
	msgReportDetailsTooLong   = "details must be at most 2000 characters"

	// PricingID
	msgRequiredPricingID = "pricingId is required"
	msgInvalidPricingID  = "pricingId is invalid"
	msgPricingGetFailed  = "failed to get pricing"

	// PolicyID
	msgInvalidPolicyID  = "policyId is invalid"
	msgRequiredPolicyID = "policyId is required"

	// Review
	msgInvalidReview      = "invalid review"
	msgCreateReviewFailed = "failed to create review"
	msgGetReviewsFailed   = "failed to get reviews"

	// Tags
	msgGetTagsFailed = "failed to get tags"

	// Recipe
	msgGetRecipeSpecFailed      = "failed to get recipe specification"
	msgGetRecipeSuggestFailed   = "failed to get recipe suggest"
	msgGetRecipeFunctionsFailed = "failed to get recipe functions"

	// Vault
	msgVaultPublicKeyGetFailed = "failed to get vault_public_key"
	msgVaultShareDeleteFailed  = "failed to delete vault share"
	msgVaultNotFound           = "vault not found"

	// Fees
	msgGetFeesFailed           = "failed to get fees"
	msgMarkFeesCollectedFailed = "failed to mark fees as collected"

	// Policy
	msgInvalidPluginPolicy    = "plugin policy is invalid"
	msgInvalidPolicySignature = "invalid policy signature"
	msgPoliciesGetFailed      = "failed to get policies"
	msgPolicyGetFailed        = "failed to get policy"
	msgPolicyCreateFailed     = "failed to create policy"
	msgPolicyDeleteFailed     = "failed to delete policy"
	msgPoliciesDeleteFailed   = "failed to delete plugin policies"
	msgPolicyEnded            = "policy has ended"

	// Signing
	msgNoMessagesToSign = "no messages to sign"
	msgTxNotAllowed     = "tx not allowed to execute"

	msgGetTxsByPolicyIDFailed = "failed to get txs by policyID"
	msgGetTxsByPluginIDFailed = "failed to get txs by pluginID"

	// Reshare
	msgReshareQueueFailed = "failed to queue reshare task"

	// Public key
	msgRequiredPublicKey      = "publicKeyECDSA is required"
	msgInvalidPublicKey       = "invalid publicKeyECDSA"
	msgInvalidPublicKeyFormat = "invalid public key format"
	msgPublicKeyMismatch      = "public key mismatch"

	// Misc. params
	msgInvalidSort       = "invalid sort parameter"
	msgInvalidPagination = "invalid pagination parameters"
	msgRequiredTaskID    = "taskId is required"

	// Message
	msgInvalidMessageFormat = "invalid message format"
	msgMessageExpired       = "message has expired"

	// Nonce
	msgNonceStoreFailed      = "failed to store nonce"
	msgNonceUsed             = "nonce already used"
	msgNonceOrExpiryRequired = "nonce and expiry time are required"
	msgExpiryTooFarInFuture  = "expiry time too far in the future"

	// Transaction
	msgTransactionCommitFailed = "failed to commit transaction"
	msgTransactionBeginFailed  = "failed to begin transaction"

	// Session
	msgStoreSessionFailed = "failed to store session"

	// Signature
	msgInvalidSignature            = "invalid signature"
	msgInvalidSignatureFormat      = "invalid signature format"
	msgSignatureVerificationFailed = "signature verification failed"

	// Proposed plugins
	msgProposedPluginValidateFailed = "failed to validate proposed plugin"
	msgProposedPluginNotApproved    = "proposed plugin not found or not approved"

	// General
	msgInvalidRequestFormat    = "invalid request format"
	msgRequestValidationFailed = "request validation failed"
	msgRequestProcessFailed    = "failed to process request"
	msgRequestParseFailed      = "failed to parse request"
	msgIssueCreditFailed       = "failed to issue credit"
	msgGetUserFeesFailed       = "failed to get user fees"
	msgGetUserTrialInfo        = "failed to get trial info"
	msgProtoMarshalFailed      = "failed to proto marshal"
	msgJSONMarshalFailed       = "failed to json marshal"
)

func NewErrorResponseWithMessage(message string) APIResponse[interface{}] {
	return APIResponse[interface{}]{
		Error: ErrorResponse{
			Message: message,
		},
		Timestamp: time.Now().Format(time.RFC3339),
		Version:   "1.0.0",
	}
}

func NewSuccessResponse[T any](code int, data T) APIResponse[T] {
	return APIResponse[T]{
		Status:    code,
		Data:      data,
		Timestamp: time.Now().Format(time.RFC3339),
		Version:   "1.0.0",
	}
}
