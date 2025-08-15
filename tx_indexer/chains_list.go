package tx_indexer

import (
	"context"
	"fmt"

	"github.com/vultisig/recipes/bitcoin"
	"github.com/vultisig/recipes/ethereum"
	rtypes "github.com/vultisig/recipes/types"
	"github.com/vultisig/verifier/tx_indexer/pkg/config"
	"github.com/vultisig/verifier/tx_indexer/pkg/rpc"
	"github.com/vultisig/vultisig-go/common"
)

// Single place to add new chains for tx indexer

type (
	SupportedChains map[common.Chain]rtypes.Chain
	SupportedRpcs   map[common.Chain]rpc.Rpc
)

func Rpcs(ctx context.Context, config config.RpcConfig) (SupportedRpcs, error) {
	btcRpc, err := rpc.NewBitcoin(config.Bitcoin.URL)
	if err != nil {
		return nil, fmt.Errorf("rpc.NewBitcoin: %w", err)
	}

	ethRpc, err := rpc.NewEvm(ctx, config.Ethereum.URL)
	if err != nil {
		return nil, fmt.Errorf("rpc.NewEvm: %w", err)
	}

	return map[common.Chain]rpc.Rpc{
		common.Bitcoin:  btcRpc,
		common.Ethereum: ethRpc,
	}, nil
}

func Chains() SupportedChains {
	return map[common.Chain]rtypes.Chain{
		common.Bitcoin:  bitcoin.NewChain(),
		common.Ethereum: ethereum.NewEthereum(),
	}
}
