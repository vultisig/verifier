package chain

import (
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/vultisig/mobile-tss-lib/tss"
	"github.com/vultisig/recipes/sdk/xrpl"
	xrpgo "github.com/xyield/xrpl-go/binary-codec"
)

type XRPIndexer struct {
	sdk *xrpl.SDK
}

func NewXRPIndexer(sdk *xrpl.SDK) *XRPIndexer {
	return &XRPIndexer{
		sdk: sdk,
	}
}

func (x *XRPIndexer) ComputeTxHash(proposedTx []byte, sigs map[string]tss.KeysignResponse) (string, error) {
	if len(sigs) == 0 {
		return "", fmt.Errorf("no signatures provided")
	}

	// For XRP, we need to extract the public key from the transaction
	// First, decode the unsigned transaction to get the SigningPubKey if present
	pubKey, err := x.extractPublicKeyFromTx(proposedTx)
	if err != nil {
		return "", fmt.Errorf("failed to extract public key from transaction: %w", err)
	}

	// Sign the transaction using the XRPL SDK
	signed, err := x.sdk.Sign(proposedTx, sigs, pubKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign transaction: %w", err)
	}

	// Extract the transaction hash from the signed transaction
	txHash, err := x.computeHashFromSignedTx(signed)
	if err != nil {
		return "", fmt.Errorf("failed to compute hash from signed transaction: %w", err)
	}

	return txHash, nil
}

// extractPublicKeyFromTx extracts the public key from the transaction data
// In XRP transactions, the public key should be provided as part of the transaction context
// For now, we'll look for it in the transaction metadata or expect it to be derivable
func (x *XRPIndexer) extractPublicKeyFromTx(txBytes []byte) ([]byte, error) {
	// Convert bytes to hex for the binary codec
	txHex := hex.EncodeToString(txBytes)
	
	// Try to decode the transaction to see if it has any key information
	txMap, err := xrpgo.Decode(strings.ToUpper(txHex))
	if err != nil {
		return nil, fmt.Errorf("failed to decode transaction: %w", err)
	}
	
	// Check if SigningPubKey is already present in the transaction
	if pubKeyHex, exists := txMap["SigningPubKey"]; exists {
		if pubKeyStr, ok := pubKeyHex.(string); ok && pubKeyStr != "" {
			pubKeyBytes, err := hex.DecodeString(pubKeyStr)
			if err != nil {
				return nil, fmt.Errorf("failed to decode SigningPubKey: %w", err)
			}
			return pubKeyBytes, nil
		}
	}
	
	// If no public key is found in the transaction, we have a problem
	// In a real implementation, the public key should be provided by the calling context
	// For now, return an error indicating the public key is missing
	return nil, fmt.Errorf("no public key found in transaction - public key must be provided in transaction metadata")
}

// computeHashFromSignedTx computes the transaction hash from a signed XRP transaction
func (x *XRPIndexer) computeHashFromSignedTx(signedTxBytes []byte) (string, error) {
	// Convert to hex
	signedHex := hex.EncodeToString(signedTxBytes)
	
	// Decode the signed transaction
	txMap, err := xrpgo.Decode(strings.ToUpper(signedHex))
	if err != nil {
		return "", fmt.Errorf("failed to decode signed transaction: %w", err)
	}
	
	// For XRP, the transaction hash is computed by taking SHA512-half of the signed transaction bytes
	// But we need to exclude the TxnSignature field for the hash computation
	txMapForHash := make(map[string]interface{})
	for k, v := range txMap {
		if k != "TxnSignature" {
			txMapForHash[k] = v
		}
	}
	
	// Re-encode without the signature
	hashableHex, err := xrpgo.Encode(txMapForHash)
	if err != nil {
		return "", fmt.Errorf("failed to encode transaction for hashing: %w", err)
	}
	
	hashableBytes, err := hex.DecodeString(hashableHex)
	if err != nil {
		return "", fmt.Errorf("failed to decode hashable hex: %w", err)
	}
	
	// Compute SHA512-half (this is how XRP computes transaction hashes)
	hash := sha512Half(hashableBytes)
	
	// Return as uppercase hex string (XRP convention)
	return strings.ToUpper(hex.EncodeToString(hash)), nil
}

// sha512Half computes SHA512 and returns first 32 bytes (same as in xrpl SDK)
func sha512Half(b []byte) []byte {
	h := sha512.Sum512(b)
	return h[:32]
}