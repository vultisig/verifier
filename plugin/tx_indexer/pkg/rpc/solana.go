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

func (s *Solana) GetTxStatus(ctx context.Context, txHash string) (TxOnChainStatus, error) {
	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	sig, err := solana.SignatureFromBase58(txHash)
	if err != nil {
		return "", fmt.Errorf("solana.SignatureFromBase58: %w", err)
	}

	tx, err := s.client.GetTransaction(ctx, sig, &rpc.GetTransactionOpts{
		Encoding: solana.EncodingBase64,
	})
	if err != nil {
		// If rate limited (429), wait before returning to slow down polling
		if strings.Contains(err.Error(), "429") || strings.Contains(err.Error(), "Too many requests") {
			time.Sleep(5 * time.Second)
		}
		return TxOnChainPending, nil
	}

	if tx == nil || tx.Meta == nil {
		return TxOnChainPending, nil
	}

	if tx.Meta.Err != nil {
		return TxOnChainFail, nil
	}

	return TxOnChainSuccess, nil
}
