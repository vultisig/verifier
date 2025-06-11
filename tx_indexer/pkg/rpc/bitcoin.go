package rpc

import (
	"context"
	"fmt"
	"strings"

	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/rpcclient"
)

type Bitcoin struct {
	client *rpcclient.Client
}

func NewBitcoin(rpcURL string) (*Bitcoin, error) {
	cl, err := rpcclient.New(&rpcclient.ConnConfig{
		Host:         strings.TrimPrefix(rpcURL, "https://"),
		HTTPPostMode: true,

		// should be not empty, otherwise btc-client would try to load cookie from a local file which is empty
		User: "user",
		Pass: "pass",
	}, nil)
	if err != nil {
		return nil, fmt.Errorf("rpcclient.New: %w", err)
	}

	// ping
	_, err = cl.GetBlockCount()
	if err != nil {
		return nil, fmt.Errorf("cl.GetBlockCount: %w", err)
	}

	return &Bitcoin{
		client: cl,
	}, nil
}

func (r *Bitcoin) GetTxStatus(ctx context.Context, txHash string) (TxOnChainStatus, error) {
	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	hash, err := chainhash.NewHashFromStr(txHash)
	if err != nil {
		return "", fmt.Errorf("chainhash.NewHashFromStr: %w", err)
	}

	tx, err := r.client.GetRawTransactionVerbose(hash)
	noConfirmations := tx != nil && tx.Confirmations == 0
	if err != nil || tx == nil || noConfirmations {
		return TxOnChainPending, nil
	}
	return TxOnChainSuccess, nil
}
