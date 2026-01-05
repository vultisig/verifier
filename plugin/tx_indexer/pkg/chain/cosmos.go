package chain

import (
	"fmt"

	"github.com/vultisig/mobile-tss-lib/tss"
	"github.com/vultisig/recipes/sdk/cosmos"
)

// CosmosIndexer handles Cosmos-based chain transaction hash computation for the tx_indexer.
// It uses the recipes SDK for all Cosmos-specific logic.
type CosmosIndexer struct {
	sdk *cosmos.SDK
}

// NewCosmosIndexer creates a new CosmosIndexer instance with the provided SDK.
func NewCosmosIndexer(sdk *cosmos.SDK) *CosmosIndexer {
	return &CosmosIndexer{
		sdk: sdk,
	}
}

// ComputeTxHash computes the transaction hash for a signed Cosmos transaction.
// The proposedTx should contain the unsigned transaction bytes (protobuf-encoded tx.Tx).
// The sigs map contains TSS signatures keyed by message hash.
// pubKey should be extracted from the proposedTx metadata or provided separately.
func (c *CosmosIndexer) ComputeTxHash(proposedTx []byte, sigs map[string]tss.KeysignResponse) (string, error) {
	if len(sigs) == 0 {
		return "", fmt.Errorf("no signatures provided")
	}

	// Extract the public key from the unsigned transaction metadata
	// For Cosmos, the pubkey should be embedded in the transaction's SignerInfo
	pubKey, err := extractCosmosPubKeyFromTx(proposedTx)
	if err != nil {
		return "", fmt.Errorf("failed to extract public key from transaction: %w", err)
	}

	// Sign the transaction using the Cosmos SDK
	signed, err := c.sdk.Sign(proposedTx, sigs, pubKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign transaction: %w", err)
	}

	// Compute the transaction hash (SHA256 of signed bytes, uppercase hex)
	return c.sdk.ComputeTxHash(signed), nil
}

// extractCosmosPubKeyFromTx attempts to extract the public key from a Cosmos transaction.
// The public key may be embedded in the SignerInfo or in a custom metadata field.
func extractCosmosPubKeyFromTx(txBytes []byte) ([]byte, error) {
	// Try to unmarshal and extract the pubkey from the transaction
	// For now, we expect the caller to embed the pubkey in a metadata envelope
	// Format: [33-byte pubkey][rest of transaction bytes]
	if len(txBytes) < 33 {
		return nil, fmt.Errorf("transaction too short to contain public key")
	}

	// Check if the first byte looks like a compressed secp256k1 pubkey (0x02 or 0x03)
	if txBytes[0] == 0x02 || txBytes[0] == 0x03 {
		return txBytes[:33], nil
	}

	return nil, fmt.Errorf("no public key found in transaction - expected 33-byte compressed key prefix")
}

