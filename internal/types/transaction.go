package types

import (
	tx_indexer_storage "github.com/vultisig/verifier/tx_indexer/pkg/storage"
)

type TransactionHistoryPaginatedList struct {
	History    []tx_indexer_storage.Tx `json:"history"`
	TotalCount uint32                  `json:"total_count"`
}
