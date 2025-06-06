package rpc

import (
	"context"
	"github.com/vultisig/verifier/internal/types"
	"time"
)

type TxIndexer interface {
	GetTxStatus(ctx context.Context, txHash string) (types.TxOnChainStatus, error)
}

const defaultTimeout = 30 * time.Second
