package rpc

import (
	"context"
	"fmt"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/vultisig/verifier/internal/types"
)

type BitcoinClient struct {
	rpc *rpcclient.Client
}

func NewBitcoinClient(rpcURL string) (*BitcoinClient, error) {
	cl, err := rpcclient.New(&rpcclient.ConnConfig{
		Host: rpcURL,
	}, nil)
	if err != nil {
		return nil, fmt.Errorf("rpcclient.New: %w", err)
	}

	return &BitcoinClient{
		rpc: cl,
	}, nil
}

func (c *BitcoinClient) GetTxStatus(ctx context.Context, txHash string) (types.TxOnChainStatus, error) {
	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	hash, err := chainhash.NewHashFromStr(txHash)
	if err != nil {
		return "", fmt.Errorf("chainhash.NewHashFromStr: %w", err)
	}

	tx, err := c.rpc.GetTransaction(hash)
	if err != nil || tx == nil {
		return types.TxOnChainPending, nil
	}
	return types.TxOnChainSuccess, nil
}
