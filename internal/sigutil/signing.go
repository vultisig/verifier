package sigutil

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// VerifyEthAddressSignature verifies if a message was signed by the owner of an Ethereum address
func VerifyEthAddressSignature(address common.Address, messageBytes []byte, signatureBytes []byte) (bool, error) {
	// Ensure signature is 65 bytes long (r, s, v)
	if len(signatureBytes) != 65 {
		return false, fmt.Errorf("invalid signature length: expected 65 bytes, got %d", len(signatureBytes))
	}

	sigCopy := make([]byte, 65)
	copy(sigCopy, signatureBytes)

	if sigCopy[64] >= 27 {
		sigCopy[64] -= 27
	}

	// Create the Ethereum prefixed message hash
	prefixedMessage := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(messageBytes), messageBytes)
	prefixedHash := crypto.Keccak256Hash([]byte(prefixedMessage))

	// Recover public key from signature
	pubkeyBytes, err := crypto.Ecrecover(prefixedHash.Bytes(), sigCopy)
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
