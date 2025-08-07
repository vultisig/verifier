package rpc

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

const defaultTimeout = 30 * time.Second

type Evm struct {
	client *ethclient.Client
}

func NewEvm(c context.Context, rpcURL string) (*Evm, error) {
	ctx, cancel := context.WithTimeout(c, defaultTimeout)
	defer cancel()

	cl, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		return nil, fmt.Errorf("ethclient.DialContext: %w", err)
	}

	return &Evm{
		client: cl,
	}, nil
}

func (r *Evm) GetTxStatus(ct context.Context, txHash string) (TxOnChainStatus, error) {
	ctx, cancel := context.WithTimeout(ct, defaultTimeout)
	defer cancel()

	rec, err := r.client.TransactionReceipt(ctx, common.HexToHash(txHash))
	if err != nil {
		if errors.Is(err, ethereum.NotFound) {
			return TxOnChainPending, nil
		}
		return "", fmt.Errorf("r.client.TransactionReceipt: %w", err)
	}
	switch rec.Status {
	case 0:
		return TxOnChainFail, nil
	case 1:
		return TxOnChainSuccess, nil
	default:
		return "", errors.New("r.client.TransactionReceipt: unknown tx receipt status by hash=" + txHash)
	}
}
