package rpc

import (
	"context"
	"strings"
)

const maxErrorMessageLength = 2048

type Rpc interface {
	GetTxStatus(ctx context.Context, txHash string) (TxStatusResult, error)
}

type TxOnChainStatus string

const (
	TxOnChainPending TxOnChainStatus = "PENDING"
	TxOnChainSuccess TxOnChainStatus = "SUCCESS"
	TxOnChainFail    TxOnChainStatus = "FAIL"
)

type TxStatusResult struct {
	Status       TxOnChainStatus
	ErrorMessage string
}

func NewTxStatusResult(status TxOnChainStatus, errorMessage string) TxStatusResult {
	errorMessage = strings.TrimSpace(errorMessage)
	if len(errorMessage) > maxErrorMessageLength {
		errorMessage = errorMessage[:maxErrorMessageLength]
	}
	return TxStatusResult{
		Status:       status,
		ErrorMessage: errorMessage,
	}
}
