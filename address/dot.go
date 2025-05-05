package address

import (
	"encoding/hex"
	"fmt"
)

func GetDotAddress(hexPublicKey string) (string, error) {
	pubKeyBytes, err := hex.DecodeString(hexPublicKey)
	if err != nil {
		return "", fmt.Errorf("invalid derived EdDSA public key: %w", err)
	}
	return SS58Encode(pubKeyBytes, 0)
}
