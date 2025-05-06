package clientutil

import (
	"encoding/hex"
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

	// Append "1" to the public key, as done in the client-side implementation
	messageToSign := publicKeyTrimmed + "1"

	// Convert to hex string with 0x prefix
	return "0x" + messageToSign
}

// ToHex converts a string to a hex string (this mimics the client-side toHex function)
func ToHex(input string) string {
	bytes := []byte(input)
	return "0x" + hex.EncodeToString(bytes)
}

// ValidateAuthRequest checks if an authentication request has all required fields
func ValidateAuthRequest(message, signature, publicKey, derivePath, chainCodeHex string) error {
	if message == "" {
		return fmt.Errorf("message is required")
	}
	if signature == "" {
		return fmt.Errorf("signature is required")
	}
	if publicKey == "" {
		return fmt.Errorf("public key is required")
	}
	if derivePath == "" {
		return fmt.Errorf("derive path is required")
	}
	if chainCodeHex == "" {
		return fmt.Errorf("chain code hex is required")
	}
	return nil
}

// FormatAuthResponse formats an authentication response according to the client expectation
func FormatAuthResponse(token string, err error) map[string]interface{} {
	if err != nil {
		return map[string]interface{}{
			"error": err.Error(),
		}
	}
	return map[string]interface{}{
		"token": token,
	}
}

// ExtractBearerToken extracts the token from the Authorization header
func ExtractBearerToken(authHeader string) (string, error) {
	if authHeader == "" {
		return "", fmt.Errorf("missing authorization header")
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return "", fmt.Errorf("invalid authorization format, use: Bearer <token>")
	}

	return parts[1], nil
}
