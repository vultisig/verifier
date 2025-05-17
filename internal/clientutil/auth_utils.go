package clientutil

import (
	"fmt"
	"strings"
)

// GenerateHexMessage creates the same message format that the client uses
// This ensures that the server can verify signatures using the same message format
func GenerateHexMessage(publicKey string) string {
	// Trim 0x prefix if present
	publicKeyTrimmed := publicKey
	if strings.HasPrefix(publicKeyTrimmed, "0x") {
		publicKeyTrimmed = publicKeyTrimmed[2:]
	}

	// Append "01" to the public key, as done in the client-side implementation
	messageToSign := publicKeyTrimmed + "01"

	// Convert to hex string with 0x prefix
	return "0x" + messageToSign
}

// ValidateAuthRequest checks if an authentication request has all required fields
func ValidateAuthRequest(message, signature, publicKey, chainCodeHex string) error {
	if message == "" {
		return fmt.Errorf("message is required")
	}
	if signature == "" {
		return fmt.Errorf("signature is required")
	}
	if publicKey == "" {
		return fmt.Errorf("public key is required")
	}
	if chainCodeHex == "" {
		return fmt.Errorf("chain code hex is required")
	}

	return nil
}
