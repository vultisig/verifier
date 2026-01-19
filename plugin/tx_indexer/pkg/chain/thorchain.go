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

func (t *THORChainIndexer) ComputeTxHash(proposedTx []byte, sigs map[string]tss.KeysignResponse) (string, error) {
	if t.sdk == nil {
		return "", fmt.Errorf("sdk not initialized")
	}
	return t.sdk.ComputeTxHash(proposedTx), nil
}
