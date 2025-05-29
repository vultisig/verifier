package rpc

import (
	"context"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/vultisig/verifier/internal/types"
)

type EvmClient struct {
	rpc *ethclient.Client
}

func NewEvmClient(c context.Context, rpcURL string) (*EvmClient, error) {
	ctx, cancel := context.WithTimeout(c, defaultTimeout)
	defer cancel()

	cl, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		return nil, fmt.Errorf("ethclient.DialContext: %w", err)
	}

	return &EvmClient{
		rpc: cl,
	}, nil
}

func (c *EvmClient) GetTxStatus(ct context.Context, txHash string) (types.TxOnChainStatus, error) {
	ctx, cancel := context.WithTimeout(ct, defaultTimeout)
	defer cancel()

	rec, err := c.rpc.TransactionReceipt(ctx, common.HexToHash(txHash))
	if err != nil {
		if errors.Is(err, ethereum.NotFound) {
			return types.TxOnChainPending, nil
		}
		return "", fmt.Errorf("c.rpc.TransactionReceipt: %w", err)
	}
	switch rec.Status {
	case 0:
		return types.TxOnChainFail, nil
	case 1:
		return types.TxOnChainSuccess, nil
	default:
		return "", errors.New("c.rpc.TransactionReceipt: unknown tx receipt status by hash=" + txHash)
	}
}
