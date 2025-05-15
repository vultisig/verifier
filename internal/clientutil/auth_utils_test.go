package clientutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateHexMessage(t *testing.T) {
	tests := []struct {
		name      string
		publicKey string
		want      string
	}{
		{
			name:      "Public key with 0x prefix",
			publicKey: "0x1234567890abcdef",
			want:      "0x1234567890abcdef01",
		},
		{
			name:      "Public key without 0x prefix",
			publicKey: "1234567890abcdef",
			want:      "0x1234567890abcdef01",
		},
		{
			name:      "Empty public key",
			publicKey: "",
			want:      "0x01",
		},
		{
			name:      "Short public key",
			publicKey: "0x123",
			want:      "0x12301",
		},
		{
			name:      "Very long public key",
			publicKey: "0x" + "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef",
			want:      "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef01",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GenerateHexMessage(tt.publicKey)
			assert.Equal(t, tt.want, got)
		})
	}
}
