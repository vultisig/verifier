package sigutil

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// VerifySignature verifies a signature against a message using a public key
func VerifySignature(vaultPublicKey string, message string, signatureBytes []byte) (bool, error) {

	// 1) Normalize pubkey
	if !strings.HasPrefix(vaultPublicKey, "0x") {
		vaultPublicKey = "0x" + vaultPublicKey
	}

	// Ensure signature is 65 bytes long (r, s, v)
	if len(signatureBytes) != 65 {
		return false, fmt.Errorf("invalid signature length: expected 65 bytes, got %d", len(signatureBytes))
	}

	// Ethereum’s v can be {0,1,27,28,…}. Shift to {27,28} as required by go-ethereum.
	if signatureBytes[64] >= 27 {
		signatureBytes[64] -= 27
	}

	// Reconstruct the exact prefixed message:
	msgBytes := []byte(message) // <-- literal UTF-8 bytes of "0x8af37…01"
	prefix := fmt.Sprintf("\x19Ethereum Signed Message:\n%d", len(msgBytes))
	prefixed := append([]byte(prefix), msgBytes...)
	digest := crypto.Keccak256Hash(prefixed)

	// fmt.Println("digest", digest.Hex())

	// Recover public key from signature
	pubkeyBytes, err := crypto.Ecrecover(digest.Bytes(), signatureBytes)
	if err != nil {
		return false, fmt.Errorf("failed to recover public key: %w", err)
	}

	// Convert recovered pubkey to address format for comparison
	recoveredPubKey, err := crypto.UnmarshalPubkey(pubkeyBytes)
	if err != nil {
		return false, fmt.Errorf("failed to unmarshal recovered public key: %w", err)
	}

	// Get Ethereum addresses from public keys
	recoveredAddr := crypto.PubkeyToAddress(*recoveredPubKey)

	// Compare addresses
	return strings.EqualFold(recoveredAddr.Hex(), vaultPublicKey), nil
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

// VerifyEthAddressSignature verifies if a message was signed by the owner of an Ethereum address
func VerifyEthAddressSignature(address common.Address, messageBytes []byte, signatureBytes []byte) (bool, error) {
	// Ensure signature is 65 bytes long (r, s, v)
	if len(signatureBytes) != 65 {
		return false, fmt.Errorf("invalid signature length: expected 65 bytes, got %d", len(signatureBytes))
	}

	// Create the Ethereum prefixed message hash
	prefixedMessage := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(messageBytes), messageBytes)
	prefixedHash := crypto.Keccak256Hash([]byte(prefixedMessage))

	// Recover public key from signature
	pubkeyBytes, err := crypto.Ecrecover(prefixedHash.Bytes(), signatureBytes)
	if err != nil {
		return false, fmt.Errorf("failed to recover public key: %w", err)
	}

	// Convert recovered pubkey to address
	recoveredPubKey, err := crypto.UnmarshalPubkey(pubkeyBytes)
	if err != nil {
		return false, fmt.Errorf("failed to unmarshal recovered public key: %w", err)
	}
	recoveredAddr := crypto.PubkeyToAddress(*recoveredPubKey)

	// Compare addresses
	return address == recoveredAddr, nil
}
