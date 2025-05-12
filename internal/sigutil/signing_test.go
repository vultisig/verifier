package sigutil

import (
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
)

func TestVerifyEthAddressSignature(t *testing.T) {
	// Generate a test key pair
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}
	address := crypto.PubkeyToAddress(privateKey.PublicKey)

	// Create a test message
	message := []byte("test message")

	// Sign the message using the standard Ethereum personal_sign method
	hash := crypto.Keccak256Hash([]byte(fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(message), message)))
	signature, err := crypto.Sign(hash.Bytes(), privateKey)
	if err != nil {
		t.Fatalf("Failed to sign message: %v", err)
	}

	tests := []struct {
		name          string
		address       common.Address
		message       []byte
		signature     []byte
		want          bool
		expectError   bool
		errorContains string
	}{
		{
			name:        "Valid signature",
			address:     address,
			message:     message,
			signature:   signature,
			want:        true,
			expectError: false,
		},
		{
			name:          "Invalid signature length",
			address:       address,
			message:       message,
			signature:     []byte{1, 2, 3}, // Too short
			want:          false,
			expectError:   true,
			errorContains: "invalid signature length",
		},
		{
			name:          "Invalid signature bytes",
			address:       address,
			message:       message,
			signature:     make([]byte, 65), // Valid length but invalid content
			want:          false,
			expectError:   true,
			errorContains: "failed to recover public key",
		},
		{
			name:        "Wrong address",
			address:     common.HexToAddress("0x1234567890123456789012345678901234567890"),
			message:     message,
			signature:   signature,
			want:        false,
			expectError: false,
		},
		{
			name:        "Empty message",
			address:     address,
			message:     []byte{},
			signature:   signature,
			want:        false,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := VerifyEthAddressSignature(tt.address, tt.message, tt.signature)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.want, got)
		})
	}
}
