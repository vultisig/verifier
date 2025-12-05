package tx_indexer

import (
	"context"
	"fmt"

	"github.com/vultisig/recipes/sdk/btc"
	"github.com/vultisig/recipes/sdk/solana"
	"github.com/vultisig/recipes/sdk/xrpl"
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

	if cfg.Solana.URL != "" {
		solRpc, err := rpc.NewSolana(cfg.Solana.URL)
		if err != nil {
			return nil, fmt.Errorf("failed to create Solana RPC client: %w", err)
		}
		rpcs[common.Solana] = solRpc
	}

	if cfg.XRP.URL != "" {
		xrpRpc, err := rpc.NewXRP(cfg.XRP.URL)
		if err != nil {
			return nil, fmt.Errorf("failed to create XRP RPC client: %w", err)
		}
		rpcs[common.XRP] = xrpRpc
	}

	if cfg.Zcash.URL != "" {
		zcashRpc, err := rpc.NewZcash(cfg.Zcash.URL)
		if err != nil {
			return nil, fmt.Errorf("failed to create Zcash RPC client: %w", err)
		}
		rpcs[common.Zcash] = zcashRpc
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

	chains[common.Bitcoin] = chain.NewBitcoinIndexer(btc.NewSDK(
		nil,
	))

	chains[common.Solana] = chain.NewSolanaIndexer(solana.NewSDK(
		nil,
	))

	chains[common.XRP] = chain.NewXRPIndexer(xrpl.NewSDK(
		nil,
	))

	chains[common.Zcash] = chain.NewZcashIndexer()

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
