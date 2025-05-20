package sigutil

import (
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func VerifyPolicySignature(ethPublicKey string, messageBytes []byte, signatureBytes []byte) (bool, error) {
	// Create the Ethereum prefixed message hash
	prefixedMessage := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(messageBytes), messageBytes)
	prefixedHash := crypto.Keccak256Hash([]byte(prefixedMessage))
	return VerifySignature(ethPublicKey, prefixedHash[:], signatureBytes)
}
func VerifySignature(ethPublicKey string, msgHash []byte, signatureBytes []byte) (bool, error) {
	if len(signatureBytes) != 65 {
		return false, fmt.Errorf("invalid signature length: expected 65 bytes, got %d", len(signatureBytes))
	}
	publicKeyBytes, err := hex.DecodeString(ethPublicKey)
	if err != nil {
		return false, err
	}
	// Validate public key length - uncompressed keys are typically 65 bytes (with prefix)
	// or 64 bytes (without prefix), compressed are 33 bytes
	validLengths := []int{33, 64, 65}
	validLength := false
	for _, length := range validLengths {
		if len(publicKeyBytes) == length {
			validLength = true
			break
		}
	}
	if !validLength {
		return false, fmt.Errorf("invalid public key length: %d bytes", len(publicKeyBytes))
	}
	pk, err := btcec.ParsePubKey(publicKeyBytes)
	if err != nil {
		return false, err
	}

	ecdsaPubKey := ecdsa.PublicKey{
		Curve: btcec.S256(),
		X:     pk.X(),
		Y:     pk.Y(),
	}
	R := new(big.Int).SetBytes(signatureBytes[:32])
	S := new(big.Int).SetBytes(signatureBytes[32:64])
	return ecdsa.Verify(&ecdsaPubKey, msgHash, R, S), nil
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
