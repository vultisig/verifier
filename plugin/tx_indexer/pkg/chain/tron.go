package chain

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/vultisig/mobile-tss-lib/tss"
	"github.com/vultisig/recipes/sdk/tron"
)

// TronIndexer handles TRON transaction hash computation for the tx_indexer.
// It uses the recipes SDK for all TRON-specific logic.
type TronIndexer struct {
	sdk *tron.SDK
}

// NewTronIndexer creates a new TronIndexer instance with the provided SDK.
func NewTronIndexer(sdk *tron.SDK) *TronIndexer {
	return &TronIndexer{
		sdk: sdk,
	}
}

// ComputeTxHash computes the transaction hash for a TRON transaction.
// For TRON, the transaction ID is SHA256(raw_data), which is computed from the
// unsigned transaction bytes (the protobuf-serialized raw_data field).
//
// The proposedTx should contain the raw_data bytes (protobuf-serialized).
// Unlike other chains, TRON's txID is deterministic from the unsigned tx,
// but we still need the signature for verification purposes.
func (t *TronIndexer) ComputeTxHash(proposedTx []byte, sigs map[string]tss.KeysignResponse) (string, error) {
	if len(sigs) == 0 {
		return "", fmt.Errorf("no signatures provided")
	}

	// Extract public key from the transaction metadata
	pubKey, err := extractTronPubKeyFromTx(proposedTx)
	if err != nil {
		return "", fmt.Errorf("failed to extract public key from transaction: %w", err)
	}

	// Get the raw transaction data (without pubkey prefix)
	rawData := proposedTx
	if len(proposedTx) > 33 && (proposedTx[0] == 0x02 || proposedTx[0] == 0x03) {
		rawData = proposedTx[33:]
	}

	// Sign to verify the signature is valid (this also validates the transaction)
	_, err = t.sdk.Sign(rawData, sigs, pubKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign transaction: %w", err)
	}

	// For TRON, the transaction ID is simply SHA256 of the raw_data bytes
	// This is deterministic and doesn't depend on the signature
	hash := sha256.Sum256(rawData)
	return hex.EncodeToString(hash[:]), nil
}

// extractTronPubKeyFromTx attempts to extract the public key from a TRON transaction.
// The public key should be prefixed to the transaction bytes.
// Format: [33-byte or 65-byte pubkey][raw_data bytes]
func extractTronPubKeyFromTx(txBytes []byte) ([]byte, error) {
	if len(txBytes) < 33 {
		return nil, fmt.Errorf("transaction too short to contain public key")
	}

	// Check if the first byte looks like a compressed secp256k1 pubkey (0x02 or 0x03)
	if txBytes[0] == 0x02 || txBytes[0] == 0x03 {
		return txBytes[:33], nil
	}

	// Check for uncompressed pubkey (0x04)
	if txBytes[0] == 0x04 && len(txBytes) >= 65 {
		return txBytes[:65], nil
	}

	return nil, fmt.Errorf("no public key found in transaction - expected compressed (33-byte) or uncompressed (65-byte) key prefix")
}

