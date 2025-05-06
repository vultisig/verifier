package sigutil

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/vultisig/mobile-tss-lib/tss"
)

// VerifySignature verifies a signature against a message using a derived public key
func VerifySignature(vaultPublicKey string, chainCodeHex string, derivePath string, messageHex []byte, signature []byte) (bool, error) {
	// Derive the public key
	derivedKeyResponse, err := tss.GetDerivedPubKey(vaultPublicKey, chainCodeHex, derivePath, false) // false for ECDSA
	if err != nil {
		return false, fmt.Errorf("failed to derive public key: %w", err)
	}

	// Extract the public key from the derived key response
	pubKeyHex := derivedKeyResponse
	if !strings.HasPrefix(pubKeyHex, "0x") {
		pubKeyHex = "0x" + pubKeyHex
	}

	// Create the Ethereum prefixed message hash
	ethMessage := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(messageHex), messageHex)
	hash := crypto.Keccak256Hash([]byte(ethMessage))

	// Extract r, s, v from signature
	if len(signature) != 65 {
		return false, fmt.Errorf("invalid signature length: expected 65 bytes, got %d", len(signature))
	}

	// Recover the signer's public key
	pubKey, err := crypto.Ecrecover(hash.Bytes(), signature)
	if err != nil {
		return false, fmt.Errorf("failed to recover public key: %w", err)
	}

	// Convert to the same format for comparison
	recoveredPubKeyHex := hexutil.Encode(pubKey)
	derivedPubKeyBytes, err := hexutil.Decode(pubKeyHex)
	if err != nil {
		return false, fmt.Errorf("failed to decode derived public key: %w", err)
	}

	// Check if recovered public key matches the derived public key
	return strings.EqualFold(recoveredPubKeyHex, hexutil.Encode(derivedPubKeyBytes)), nil
}

// RawSignature converts r, s, v values to a raw signature byte array
func RawSignature(r *big.Int, s *big.Int, recoveryID uint8) []byte {
	var signature [65]byte
	rBytes := r.Bytes()
	sBytes := s.Bytes()

	// Ensure r and s values fill 32 bytes with leading zeros if needed
	copy(signature[32-len(rBytes):32], rBytes)
	copy(signature[64-len(sBytes):64], sBytes)

	// Set recovery ID
	signature[64] = byte(recoveryID)

	return signature[:]
}
