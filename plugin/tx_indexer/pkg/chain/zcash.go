package chain

import (
	"fmt"

	"github.com/vultisig/mobile-tss-lib/tss"
	"github.com/vultisig/recipes/sdk/zcash"
)

// ZcashIndexer handles Zcash transaction hash computation for the tx_indexer.
// It uses the recipes SDK for all Zcash-specific logic.
type ZcashIndexer struct{}

// NewZcashIndexer creates a new ZcashIndexer instance.
func NewZcashIndexer() *ZcashIndexer {
	return &ZcashIndexer{}
}

// ComputeTxHash computes the transaction hash for a signed Zcash transaction.
// The proposedTx should contain the raw transaction bytes with embedded metadata
// (created using zcash.SerializeWithMetadata from the SDK).
// The embedded metadata includes:
// - Public key (33 bytes) for building scriptSig
// - Pre-computed sighashes for signature lookup
//
// It orders signatures correctly by using the embedded sighashes to look up
// signatures from the map using the derived key (SHA256 + Base64).
func (z *ZcashIndexer) ComputeTxHash(proposedTx []byte, sigs map[string]tss.KeysignResponse, _ []byte) (string, error) {
	if len(sigs) == 0 {
		return "", fmt.Errorf("no signatures provided")
	}

	// Use the recipes SDK to sign and compute the hash
	// The SDK handles:
	// 1. Parsing the transaction and extracting embedded metadata (pubkey, sighashes)
	// 2. Looking up signatures using derived keys (SHA256 + Base64 of sighash)
	// 3. Building the signed transaction with proper scriptSig
	// 4. Computing the double SHA256 hash
	return zcash.SignAndComputeHashFromRaw(proposedTx, sigs)
}
