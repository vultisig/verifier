package tx_indexer

import (
	"context"
	"fmt"
	"github.com/vultisig/recipes/bitcoin"
	"github.com/vultisig/recipes/ethereum"
	rtypes "github.com/vultisig/recipes/types"
	"github.com/vultisig/verifier/common"
	"github.com/vultisig/verifier/config"
	vbtc "github.com/vultisig/verifier/internal/chains/bitcoin"
	"github.com/vultisig/verifier/internal/chains/evm"
	"github.com/vultisig/verifier/internal/types"
)

// Single place to add new chains for tx indexer

func Rpcs(ctx context.Context, config config.RpcConfig) (map[common.Chain]types.TxIndexerRpc, error) {
	btcRpc, err := vbtc.NewRpc(config.Bitcoin.URL)
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

func Chains() map[common.Chain]rtypes.Chain {
	return map[common.Chain]rtypes.Chain{
		common.Bitcoin:  bitcoin.NewChain(),
		common.Ethereum: ethereum.NewEthereum(),
	}
}
