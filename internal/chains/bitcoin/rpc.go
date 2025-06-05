package bitcoin

import (
	"context"
	"fmt"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/vultisig/verifier/internal/types"
	"strings"
)

type Rpc struct {
	client *rpcclient.Client
}

func NewRpc(rpcURL string) (*Rpc, error) {
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

	return &Rpc{
		client: cl,
	}, nil
}

func (r *Rpc) GetTxStatus(ctx context.Context, txHash string) (types.TxOnChainStatus, error) {
	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	hash, err := chainhash.NewHashFromStr(txHash)
	if err != nil {
		return "", fmt.Errorf("chainhash.NewHashFromStr: %w", err)
	}

	tx, err := r.client.GetRawTransaction(hash)
	if err != nil || tx == nil {
		return types.TxOnChainPending, nil
	}
	return types.TxOnChainSuccess, nil
}
