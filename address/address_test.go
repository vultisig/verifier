package address

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/vultisig/verifier/common"
)

func TestGetAddress(t *testing.T) {
	tests := []struct {
		name  string
		chain common.Chain
		want  string
	}{
		{
			name:  "BitcoinCash",
			chain: common.BitcoinCash,
			want:  "qzsvzzkwt9tjl4lv5c4zwks2nse50gqq6scda6xp00",
		},
		{
			name:  "Bitcoin",
			chain: common.Bitcoin,
			want:  "bc1qxpeg8k8xrygj9ae8q6pkzj29sf7w8e7krm4v5f",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetAddress(testECDSAPublicKey, testHexChainCode, tt.chain)
			if err != nil {
				t.Error(err)
				t.FailNow()
			}
			assert.Equal(t, got, tt.want)
		})
	}
}
