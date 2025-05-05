package address

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/vultisig/mobile-tss-lib/tss"
	"github.com/vultisig/verifier/common"
)

func TestGetAddress(t *testing.T) {
	tests := []struct {
		name     string
		chain    common.Chain
		want     string
		inputKey string
		isEdDSA  bool
	}{
		{
			name:     "BitcoinCash",
			chain:    common.BitcoinCash,
			want:     "qzsvzzkwt9tjl4lv5c4zwks2nse50gqq6scda6xp00",
			inputKey: testECDSAPublicKey,
			isEdDSA:  false,
		},
		{
			name:     "Bitcoin",
			chain:    common.Bitcoin,
			want:     "bc1qxpeg8k8xrygj9ae8q6pkzj29sf7w8e7krm4v5f",
			inputKey: testECDSAPublicKey,
			isEdDSA:  false,
		},
		{
			name:     "Sui",
			chain:    common.Sui,
			want:     "0x7a4629f9194d10526e80d76be734535bd5581ef37760d6914052d26066a8ff7b",
			inputKey: testEdDSAPublicKey,
			isEdDSA:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, gotPublicKey, gotIsEdDSA, err := GetAddress(tt.inputKey, testHexChainCode, tt.chain)
			if err != nil {
				t.Error(err)
				t.FailNow()
			}
			assert.Equal(t, tt.want, got)
			// We don't support deriving the public key for EdDSA chains
			if !tt.isEdDSA {
				expectedPublicKey, err := tss.GetDerivedPubKey(tt.inputKey, testHexChainCode, tt.chain.GetDerivePath(), tt.chain.IsEdDSA())
				if err != nil {
					t.Error(err)
					t.FailNow()
				}
				assert.Equal(t, expectedPublicKey, gotPublicKey)
			}
			assert.Equal(t, tt.isEdDSA, gotIsEdDSA)
		})
	}
}
