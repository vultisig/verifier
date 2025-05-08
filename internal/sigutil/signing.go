package sigutil

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/crypto"
)

// VerifySignature verifies a signature against a message using a derived public key
// This is a placeholder that would be implemented with actual TSS library integration
func VerifySignature(vaultPublicKey string, chainCodeHex string, messageHex []byte, signature []byte) (bool, error) {
	// In a real implementation, this would:
	// 1. Derive the public key using TSS library's GetDerivedPubKey function
	// 2. Parse the public key
	// 3. Verify the signature against the message hash
	// 4. Return the verification result

	// Placeholder implementation for verification logic
	// This would be replaced with actual TSS integration

	// Hash the message with Ethereum prefix - just demonstrating the pattern
	_ = crypto.Keccak256([]byte(fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(messageHex), messageHex)))

	// In the real implementation, this would derive and verify with the actual key
	// For now, we'll return an error to indicate this needs implementation
	return false, fmt.Errorf("not implemented: requires TSS library integration")
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
