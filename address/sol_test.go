package address

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/vultisig/verifier/common"
)

func TestGetSolAddress(t *testing.T) {
	tests := []struct {
		name  string
		chain common.Chain
		want  string
	}{
		{
			name:  "Solana",
			chain: common.Solana,
			want:  "46ZJUzqDR1dxvX7hFWogsAzyAseAwtb1XNGhtCCNCHW5",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetSolAddress(testEdDSAPublicKey)
			if err != nil {
				t.Error(err)
				t.FailNow()
			}
			assert.Equal(t, tt.want, got)
		})
	}
}
