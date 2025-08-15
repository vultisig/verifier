package tx_indexer

import (
	"context"
	"fmt"

	"github.com/vultisig/verifier/plugin/tx_indexer/pkg/chain"
	"github.com/vultisig/verifier/plugin/tx_indexer/pkg/config"
	"github.com/vultisig/verifier/plugin/tx_indexer/pkg/rpc"
	"github.com/vultisig/vultisig-go/common"
)

type (
	SupportedChains map[common.Chain]chain.Indexer
	SupportedRpcs   map[common.Chain]rpc.Rpc
)

func Rpcs(ctx context.Context, cfg config.RpcConfig) (SupportedRpcs, error) {
	rpcs := make(SupportedRpcs)

	if cfg.Bitcoin.URL != "" {
		btcRpc, err := rpc.NewBitcoin(cfg.Bitcoin.URL)
		if err != nil {
			return nil, fmt.Errorf("failed to create Bitcoin RPC client: %w", err)
		}
		rpcs[common.Bitcoin] = btcRpc
	}

	evmChains := map[common.Chain]config.RpcItem{
		common.Ethereum:    cfg.Ethereum,
		common.Avalanche:   cfg.Avalanche,
		common.BscChain:    cfg.BscChain,
		common.Arbitrum:    cfg.Arbitrum,
		common.Base:        cfg.Base,
		common.Optimism:    cfg.Optimism,
		common.Polygon:     cfg.Polygon,
		common.Blast:       cfg.Blast,
		common.CronosChain: cfg.Cronos,
		common.Zksync:      cfg.Zksync,
	}

	for chainID, rpcConfig := range evmChains {
		if rpcConfig.URL != "" {
			evmRpc, err := rpc.NewEvm(ctx, rpcConfig.URL)
			if err != nil {
				return nil, fmt.Errorf("failed to create EVM RPC client for %s: %w", chainID.String(), err)
			}
			rpcs[chainID] = evmRpc
		}
	}

	return rpcs, nil
}

func Chains() (SupportedChains, error) {
	chains := make(map[common.Chain]chain.Indexer)

	chains[common.Bitcoin] = chain.NewBitcoinIndexer()

	evmChains := []common.Chain{
		common.Ethereum,
		common.Avalanche,
		common.BscChain,
		common.Arbitrum,
		common.Base,
		common.Optimism,
		common.Polygon,
		common.Blast,
		common.CronosChain,
		common.Zksync,
	}

	for _, chainType := range evmChains {
		evmID, err := chainType.EvmID()
		if err != nil {
			return nil, fmt.Errorf("failed to initialize chain %s: %w", chainType.String(), err)
		}
		chains[chainType] = chain.NewEvmIndexer(evmID)
	}

	return chains, nil
}
