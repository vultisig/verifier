package chain

import (
	"github.com/vultisig/mobile-tss-lib/tss"
	tronSDK "github.com/vultisig/recipes/sdk/tron"
)

type TronIndexer struct {
	sdk *tronSDK.SDK
}

func NewTronIndexer(sdk *tronSDK.SDK) *TronIndexer {
	return &TronIndexer{
		sdk: sdk,
	}
}

func (t *TronIndexer) ComputeTxHash(proposedTx []byte, _ map[string]tss.KeysignResponse, _ []byte) (string, error) {
	// For TRON, the transaction ID is the SHA256 hash of the raw_data bytes
	// The signatures are not needed to compute the transaction hash
	return t.sdk.ComputeTxHash(proposedTx), nil
}
