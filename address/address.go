package address

import (
	"fmt"

	"github.com/vultisig/mobile-tss-lib/tss"

	"github.com/vultisig/verifier/common"
)

// GetAddress returns the address for the given public key and chain.
func GetAddress(rootHexPublicKey string, rootChainCode string, chain common.Chain) (string, error) {
	hexPublicKey, err := tss.GetDerivedPubKey(rootHexPublicKey, rootChainCode, chain.GetDerivePath(), chain.IsEdDSA())
	if err != nil {
		return "", fmt.Errorf("failed to derive public key: %w", err)
	}
	switch chain {
	case common.Bitcoin:
		return GetBitcoinAddress(hexPublicKey)
	case common.BitcoinCash:
		return GetBitcoinCashAddress(hexPublicKey)
	case common.Litecoin:
		return GetLitecoinAddress(hexPublicKey)
	case common.GaiaChain:
		return GetBech32Address(hexPublicKey, `cosmos`)
	case common.THORChain:
		return GetBech32Address(hexPublicKey, `thor`)
	case common.MayaChain:
		return GetBech32Address(hexPublicKey, `maya`)
	case common.Kujira:
		return GetBech32Address(hexPublicKey, `kujira`)
	case common.Dydx:
		return GetBech32Address(hexPublicKey, `dydx`)
	case common.TerraClassic, common.Terra:
		return GetBech32Address(hexPublicKey, `terra`)
	case common.Osmosis:
		return GetBech32Address(hexPublicKey, `osmosis`)
	case common.Noble:
		return GetBech32Address(hexPublicKey, `noble`)
	case common.Arbitrum, common.Base, common.BscChain, common.Ethereum, common.Polygon, common.Blast, common.Avalanche, common.Optimism, common.CronosChain, common.Zksync:
		return GetEVMAddress(hexPublicKey)
	case common.Sui:
		return GetSuiAddress(hexPublicKey)
	case common.Solana:
		return GetSolAddress(hexPublicKey)
	default:
		return "", fmt.Errorf("unsupported chain: %s", chain)
	}
}
