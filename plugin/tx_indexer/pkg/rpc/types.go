package rpc

import "context"

type Rpc interface {
	GetTxStatus(ctx context.Context, txHash string) (TxOnChainStatus, error)
}

type TxOnChainStatus string

const (
	TxOnChainPending TxOnChainStatus = "PENDING"
	TxOnChainSuccess TxOnChainStatus = "SUCCESS"
	TxOnChainFail    TxOnChainStatus = "FAIL"
)
