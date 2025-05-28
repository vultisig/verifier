package tx_indexer

import (
	"context"
	"github.com/vultisig/verifier/internal/types"
	"time"
)

type Client interface {
	GetTxStatus(ctx context.Context, txHash string) (types.TxOnChainStatus, error)
}

const defaultTimeout = 30 * time.Second
