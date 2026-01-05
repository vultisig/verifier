package chain

import (
	"fmt"

	"github.com/vultisig/mobile-tss-lib/tss"
	"github.com/vultisig/recipes/sdk/cosmos"
)

// MayaIndexer handles MayaChain transaction hash computation for the tx_indexer.
// MayaChain is a Cosmos SDK based chain, so it shares the same transaction format.
type MayaIndexer struct {
	sdk *cosmos.SDK
}

// NewMayaIndexer creates a new MayaIndexer instance with the provided SDK.
func NewMayaIndexer(sdk *cosmos.SDK) *MayaIndexer {
	return &MayaIndexer{
		sdk: sdk,
	}
}

// ComputeTxHash computes the transaction hash for a signed MayaChain transaction.
// The proposedTx should contain the unsigned transaction bytes (protobuf-encoded tx.Tx).
// The sigs map contains TSS signatures keyed by message hash.
func (m *MayaIndexer) ComputeTxHash(proposedTx []byte, sigs map[string]tss.KeysignResponse) (string, error) {
	if len(sigs) == 0 {
		return "", fmt.Errorf("no signatures provided")
	}

	// Extract the public key from the unsigned transaction metadata
	pubKey, err := extractCosmosPubKeyFromTx(proposedTx)
	if err != nil {
		return "", fmt.Errorf("failed to extract public key from transaction: %w", err)
	}

	// Sign the transaction using the Cosmos SDK
	signed, err := m.sdk.Sign(proposedTx, sigs, pubKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign transaction: %w", err)
	}

	// Compute the transaction hash (SHA256 of signed bytes, uppercase hex)
	return m.sdk.ComputeTxHash(signed), nil
}

