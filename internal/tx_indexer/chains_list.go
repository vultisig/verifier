package tx_indexer

import (
	"context"
	"fmt"
	"github.com/vultisig/verifier/common"
	"github.com/vultisig/verifier/config"
	"github.com/vultisig/verifier/internal/chains/bitcoin"
	"github.com/vultisig/verifier/internal/chains/evm"
	"github.com/vultisig/verifier/internal/types"
)

// Single place to add new chains for tx indexer

func Rpc(ctx context.Context, config config.RpcConfig) (map[common.Chain]types.TxIndexerRpc, error) {
	btcRpc, err := bitcoin.NewRpc(config.Bitcoin.URL)
	if err != nil {
		return nil, fmt.Errorf("bitcoin.NewRpc: %w", err)
	}

	ethRpc, err := evm.NewRpc(ctx, config.Ethereum.URL)
	if err != nil {
		return nil, fmt.Errorf("evm.NewRpc[eth]: %w", err)
	}

	return map[common.Chain]types.TxIndexerRpc{
		common.Bitcoin:  btcRpc,
		common.Ethereum: ethRpc,
	}, nil
}

func Tss() (map[common.Chain]types.TxIndexerTss, error) {
	ethID, err := common.Ethereum.EVMChainID()
	if err != nil {
		return nil, fmt.Errorf("common.Ethereum.EVMChainID: %w", err)
	}

	return map[common.Chain]types.TxIndexerTss{
		//common.Bitcoin:  bitcoin.NewTss(),
		common.Ethereum: evm.NewTss(ethID),
	}, nil
}
