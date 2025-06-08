package evm

import (
	"context"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/vultisig/verifier/internal/types"
	"time"
)

const defaultTimeout = 30 * time.Second

type Rpc struct {
	client *ethclient.Client
}

func NewRpc(c context.Context, rpcURL string) (*Rpc, error) {
	ctx, cancel := context.WithTimeout(c, defaultTimeout)
	defer cancel()

	cl, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		return nil, fmt.Errorf("ethclient.DialContext: %w", err)
	}

	return &Rpc{
		client: cl,
	}, nil
}

func (r *Rpc) GetTxStatus(ct context.Context, txHash string) (types.TxOnChainStatus, error) {
	ctx, cancel := context.WithTimeout(ct, defaultTimeout)
	defer cancel()

	rec, err := r.client.TransactionReceipt(ctx, common.HexToHash(txHash))
	if err != nil {
		if errors.Is(err, ethereum.NotFound) {
			return types.TxOnChainPending, nil
		}
		return "", fmt.Errorf("r.client.TransactionReceipt: %w", err)
	}
	switch rec.Status {
	case 0:
		return types.TxOnChainFail, nil
	case 1:
		return types.TxOnChainSuccess, nil
	default:
		return "", errors.New("r.client.TransactionReceipt: unknown tx receipt status by hash=" + txHash)
	}
}
