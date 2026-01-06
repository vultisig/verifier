package types

import (
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/vultisig/verifier/plugin/tx_indexer/pkg/rpc"
	"github.com/vultisig/verifier/plugin/tx_indexer/pkg/storage"
	vtypes "github.com/vultisig/verifier/types"
	"github.com/vultisig/vultisig-go/common"
)

// PluginTransactionResponse is the response DTO for plugin transactions.
// It matches the Fee format as closely as possible for frontend consistency.
type PluginTransactionResponse struct {
	ID            uuid.UUID            `json:"id"`
	PluginID      vtypes.PluginID      `json:"plugin_id"`
	AppName       string               `json:"app_name"` // Plugin title for display
	PolicyID      uuid.UUID            `json:"policy_id"`
	PublicKey     string               `json:"public_key"` // Matches Fee format (was from_public_key)
	ToPublicKey   string               `json:"to_public_key"`
	Chain         common.Chain         `json:"chain"`
	TokenID       string               `json:"token_id"`
	Amount        *string              `json:"amount"` // Transaction amount in base units
	TxHash        *string              `json:"tx_hash"`
	Status        storage.TxStatus     `json:"status"`
	StatusOnChain *rpc.TxOnChainStatus `json:"status_onchain"`
	CreatedAt     time.Time            `json:"created_at"`
	UpdatedAt     time.Time            `json:"updated_at"`
	BroadcastedAt *time.Time           `json:"broadcasted_at"`
}

// FromStorageTxs converts a slice of storage.Tx to a slice of PluginTransactionResponse
// titleMap maps plugin_id to app name/title
func FromStorageTxs(txs []storage.Tx, titleMap map[string]string) []PluginTransactionResponse {
	result := make([]PluginTransactionResponse, len(txs))
	for i, tx := range txs {
		appName := titleMap[string(tx.PluginID)]
		result[i] = PluginTransactionResponse{
			ID:            tx.ID,
			PluginID:      tx.PluginID,
			AppName:       appName,
			PolicyID:      tx.PolicyID,
			PublicKey:     tx.FromPublicKey,
			ToPublicKey:   tx.ToPublicKey,
			Chain:         common.Chain(tx.ChainID),
			TokenID:       tx.TokenID,
			Amount:        tx.Amount,
			TxHash:        tx.TxHash,
			Status:        tx.Status,
			StatusOnChain: tx.StatusOnChain,
			CreatedAt:     tx.CreatedAt,
			UpdatedAt:     tx.UpdatedAt,
			BroadcastedAt: tx.BroadcastedAt,
		}
	}
	return result
}

type TransactionHistoryPaginatedList struct {
	History    []PluginTransactionResponse `json:"history"`
	TotalCount uint32                      `json:"total_count"`
}

// PluginFeeResponse is the response DTO for plugin fees.
// It matches the transaction history format for frontend consistency.
type PluginFeeResponse struct {
	ID              uint64           `json:"id"`
	PluginID        vtypes.PluginID  `json:"plugin_id"`
	AppName         string           `json:"app_name"` // Plugin title for display
	PolicyID        uuid.UUID        `json:"policy_id"`
	PublicKey       string           `json:"public_key"`
	TransactionType string           `json:"transaction_type"` // fee type: installation_fee, subscription_fee, etc.
	Amount          string           `json:"amount"`           // String for consistency with plugin history
	Status          vtypes.FeeStatus `json:"status"`
	CreatedAt       time.Time        `json:"created_at"`
}

// FeeWithStatus extends Fee with derived status from batch
type FeeWithStatus struct {
	vtypes.Fee
	Status vtypes.FeeStatus
}

// FromFeesWithStatus converts a slice of FeeWithStatus to a slice of PluginFeeResponse
// titleMap maps plugin_id to app name/title
func FromFeesWithStatus(fees []FeeWithStatus, titleMap map[string]string) []PluginFeeResponse {
	result := make([]PluginFeeResponse, len(fees))
	for i, fee := range fees {
		appName := titleMap[fee.PluginID]
		result[i] = PluginFeeResponse{
			ID:              fee.ID,
			PluginID:        vtypes.PluginID(fee.PluginID),
			AppName:         appName,
			PolicyID:        fee.PolicyID,
			PublicKey:       fee.PublicKey,
			TransactionType: fee.FeeType,
			Amount:          strconv.FormatUint(fee.Amount, 10),
			Status:          fee.Status,
			CreatedAt:       fee.CreatedAt.UTC(),
		}
	}
	return result
}

type FeeHistoryPaginatedList struct {
	History    []PluginFeeResponse `json:"history"`
	TotalCount uint32              `json:"total_count"`
}

// PluginBillingSummaryRow is the raw data from the database query
type PluginBillingSummaryRow struct {
	PluginID      string
	PricingType   string  // once, recurring, per-tx
	PricingAmount uint64  // amount in smallest unit
	PricingAsset  string  // usdc
	Frequency     *string // daily, weekly, biweekly, monthly (nil for non-recurring)
	StartDate     time.Time
	TotalFees     uint64
}

// PluginBillingSummary is the response DTO for plugin billing info
type PluginBillingSummary struct {
	PluginID    vtypes.PluginID `json:"plugin_id"`
	AppName     string          `json:"app_name"`
	PricingType string          `json:"pricing_type"` // once, recurring, per-tx
	Pricing     string          `json:"pricing"`      // Formatted: "0.01 USDC per transaction"
	StartDate   time.Time       `json:"start_date"`
	NextPayment *time.Time      `json:"next_payment"` // nil for non-recurring
	TotalFees   string          `json:"total_fees"`
}

// PluginBillingSummaryList is the response for the billing summary endpoint
type PluginBillingSummaryList struct {
	Plugins    []PluginBillingSummary `json:"plugins"`
	TotalCount uint32                 `json:"total_count"`
}
