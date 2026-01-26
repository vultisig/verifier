package chain

import (
	"fmt"

	"github.com/vultisig/mobile-tss-lib/tss"
	cosmossdk "github.com/vultisig/recipes/sdk/cosmos"
)

type MayaChainIndexer struct {
	sdk *cosmossdk.SDK
}

func NewMayaChainIndexer(sdk *cosmossdk.SDK) *MayaChainIndexer {
	return &MayaChainIndexer{
		sdk: sdk,
	}
}

func (m *MayaChainIndexer) ComputeTxHash(proposedTx []byte, sigs map[string]tss.KeysignResponse, pubKey []byte) (string, error) {
	if m.sdk == nil {
		return "", fmt.Errorf("sdk not initialized")
	}
	signedTx, err := m.sdk.Sign(proposedTx, sigs, pubKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign tx: %w", err)
	}
	return m.sdk.ComputeTxHash(signedTx), nil
}
