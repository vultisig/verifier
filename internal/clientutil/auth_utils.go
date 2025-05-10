package clientutil

import (
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
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

	// Validate hex formats
	if !strings.HasPrefix(message, "0x") {
		return fmt.Errorf("message must be a hex string with 0x prefix")
	}
	if !isValidPrefixedHex(signature) {
		return fmt.Errorf("signature must be a valid 0x-prefixed hex string")
	}
	if !isValidPrefixedHex(publicKey) {
		return fmt.Errorf("public key must be a valid 0x-prefixed hex string")
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

// StripHexPrefix removes the 0x prefix from a hex string if present
func StripHexPrefix(hex string) string {
	if strings.HasPrefix(hex, "0x") {
		return hex[2:]
	}
	return hex
}

// AddHexPrefix adds the 0x prefix to a hex string if not present
func AddHexPrefix(hex string) string {
	if !strings.HasPrefix(hex, "0x") {
		return "0x" + hex
	}
	return hex
}

// GetTimestamp returns the current Unix timestamp in seconds
func GetTimestamp() int64 {
	return time.Now().Unix()
}

// IsValidEthAddress checks if a string is a valid Ethereum address
func IsValidEthAddress(address string) bool {
	if !strings.HasPrefix(address, "0x") {
		return false
	}
	return common.IsHexAddress(address)
}

// helper
func isValidPrefixedHex(v string) bool {
	if !strings.HasPrefix(v, "0x") {
		return false
	}
	_, err := hex.DecodeString(v[2:])
	return err == nil
}
