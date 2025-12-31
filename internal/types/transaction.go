package types

import (
	"time"

	"github.com/google/uuid"
	"github.com/vultisig/verifier/plugin/tx_indexer/pkg/rpc"
	"github.com/vultisig/verifier/plugin/tx_indexer/pkg/storage"
	vtypes "github.com/vultisig/verifier/types"
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
	ChainID       int                  `json:"chain_id"`
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
func FromStorageTxs(txs []storage.Tx, appName string) []PluginTransactionResponse {
	result := make([]PluginTransactionResponse, len(txs))
	for i, tx := range txs {
		result[i] = PluginTransactionResponse{
			ID:            tx.ID,
			PluginID:      tx.PluginID,
			AppName:       appName,
			PolicyID:      tx.PolicyID,
			PublicKey:     tx.FromPublicKey,
			ToPublicKey:   tx.ToPublicKey,
			ChainID:       tx.ChainID,
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
