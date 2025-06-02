package clientutil

import (
	"fmt"
)

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
