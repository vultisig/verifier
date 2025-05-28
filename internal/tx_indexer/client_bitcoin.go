package tx_indexer

import (
	"context"
	"github.com/vultisig/verifier/internal/types"
	"time"
)

type BitcoinClient struct {
}

func NewBitcoinClient(c context.Context, rpcURL string) (*BitcoinClient, error) {
	ctx, cancel := context.WithTimeout(c, defaultTimeout)
	defer cancel()

	return &BitcoinClient{}, nil
}

func (c *BitcoinClient) GetTxStatus(ct context.Context, txHash string) (types.TxOnChainStatus, error) {
	ctx, cancel := context.WithTimeout(ct, defaultTimeout)
	defer cancel()

}
