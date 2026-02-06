package rpc

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
)

type Solana struct {
	client *rpc.Client
}

func NewSolana(rpcURL string) (*Solana, error) {
	cl := rpc.New(rpcURL)

	ctx := context.Background()
	_, err := cl.GetHealth(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to ping: %w", err)
	}

	return &Solana{
		client: cl,
	}, nil
}

func (s *Solana) GetTxStatus(ctx context.Context, txHash string) (TxStatusResult, error) {
	if ctx.Err() != nil {
		return TxStatusResult{}, ctx.Err()
	}

	sig, err := solana.SignatureFromBase58(txHash)
	if err != nil {
		return TxStatusResult{}, fmt.Errorf("solana.SignatureFromBase58: %w", err)
	}

	tx, err := s.client.GetTransaction(ctx, sig, &rpc.GetTransactionOpts{
		Encoding: solana.EncodingBase64,
	})
	if err != nil {
		if strings.Contains(err.Error(), "429") || strings.Contains(err.Error(), "Too many requests") {
			time.Sleep(5 * time.Second)
		}
		return NewTxStatusResult(TxOnChainPending, ""), nil
	}

	if tx == nil || tx.Meta == nil {
		return NewTxStatusResult(TxOnChainPending, ""), nil
	}

	if tx.Meta.Err != nil {
		errorMsg := extractSolanaErrorMessage(tx.Meta)
		return NewTxStatusResult(TxOnChainFail, errorMsg), nil
	}

	return NewTxStatusResult(TxOnChainSuccess, ""), nil
}

func extractSolanaErrorMessage(meta *rpc.TransactionMeta) string {
	if meta == nil {
		return ""
	}

	const failedMarker = " failed: "

	// Prefer human-readable "failed:" line from logs (scan backwards)
	for i := len(meta.LogMessages) - 1; i >= 0; i-- {
		log := meta.LogMessages[i]
		if idx := strings.Index(log, failedMarker); idx != -1 {
			msg := strings.TrimSpace(log[idx+len(failedMarker):])
			if msg != "" {
				return msg
			}
		}
	}

	// Fallback: structured error
	if meta.Err != nil {
		msg := strings.TrimSpace(fmt.Sprintf("%v", meta.Err))
		if msg != "" {
			return msg
		}
	}

	return ""
}
