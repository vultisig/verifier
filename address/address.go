package address

import (
	"fmt"

	"github.com/vultisig/mobile-tss-lib/tss"

	"github.com/vultisig/verifier/common"
)

// GetAddress returns the address, public key, isEdDSA, and error for the given public key and chain.
func GetAddress(rootHexPublicKey string, rootChainCode string, chain common.Chain) (address string, publicKey string, isEdDSA bool, err error) {
	if len(rootHexPublicKey) != 66 && len(rootHexPublicKey) != 64 {
		return "", "", false, fmt.Errorf("invalid public key: %s", rootHexPublicKey)
	}

	if !chain.IsEdDSA() {
		publicKey, err = tss.GetDerivedPubKey(rootHexPublicKey, rootChainCode, chain.GetDerivePath(), chain.IsEdDSA())
		if err != nil {
			return "", "", false, fmt.Errorf("failed to derive public key: %w", err)
		}
	} else {
		publicKey = rootHexPublicKey
	}

	switch chain {
	case common.Bitcoin:
		address, err = GetBitcoinAddress(publicKey)
		return address, publicKey, chain.IsEdDSA(), err
	case common.BitcoinCash:
		address, err = GetBitcoinCashAddress(publicKey)
		return address, publicKey, chain.IsEdDSA(), err
	case common.Litecoin:
		address, err = GetLitecoinAddress(publicKey)
		return address, publicKey, chain.IsEdDSA(), err
	case common.GaiaChain:
		address, err = GetBech32Address(publicKey, `cosmos`)
		return address, publicKey, chain.IsEdDSA(), err
	case common.THORChain:
		address, err = GetBech32Address(publicKey, `thor`)
		return address, publicKey, chain.IsEdDSA(), err
	case common.MayaChain:
		address, err = GetBech32Address(publicKey, `maya`)
		return address, publicKey, chain.IsEdDSA(), err
	case common.Kujira:
		address, err = GetBech32Address(publicKey, `kujira`)
		return address, publicKey, chain.IsEdDSA(), err
	case common.Dydx:
		address, err = GetBech32Address(publicKey, `dydx`)
		return address, publicKey, chain.IsEdDSA(), err
	case common.TerraClassic, common.Terra:
		address, err = GetBech32Address(publicKey, `terra`)
		return address, publicKey, chain.IsEdDSA(), err
	case common.Osmosis:
		address, err = GetBech32Address(publicKey, `osmosis`)
		return address, publicKey, chain.IsEdDSA(), err
	case common.Noble:
		address, err = GetBech32Address(publicKey, `noble`)
		return address, publicKey, chain.IsEdDSA(), err
	case common.Arbitrum, common.Base, common.BscChain, common.Ethereum, common.Polygon, common.Blast, common.Avalanche, common.Optimism, common.CronosChain, common.Zksync:
		address, err = GetEVMAddress(publicKey)
		return address, publicKey, chain.IsEdDSA(), err
	case common.Sui:
		address, err = GetSuiAddress(publicKey)
		return address, publicKey, chain.IsEdDSA(), err
	case common.Solana:
		address, err = GetSolAddress(publicKey)
		return address, publicKey, chain.IsEdDSA(), err
	default:
		return "", "", false, fmt.Errorf("unsupported chain: %s", chain)
	}
}
