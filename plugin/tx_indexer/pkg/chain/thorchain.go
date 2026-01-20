package chain

import (
	"fmt"

	"github.com/vultisig/mobile-tss-lib/tss"
	cosmossdk "github.com/vultisig/recipes/sdk/cosmos"
)

type THORChainIndexer struct {
	sdk *cosmossdk.SDK
}

func NewTHORChainIndexer(sdk *cosmossdk.SDK) *THORChainIndexer {
	return &THORChainIndexer{
		sdk: sdk,
	}
}

func (t *THORChainIndexer) ComputeTxHash(proposedTx []byte, sigs map[string]tss.KeysignResponse, pubKey []byte) (string, error) {
	if t.sdk == nil {
		return "", fmt.Errorf("sdk not initialized")
	}
	signedTx, err := t.sdk.Sign(proposedTx, sigs, pubKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign tx: %w", err)
	}
	return t.sdk.ComputeTxHash(signedTx), nil
}
