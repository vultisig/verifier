package chain

import (
	"testing"

	"github.com/vultisig/mobile-tss-lib/tss"
)

func TestTHORChainIndexer_NilSDK(t *testing.T) {
	indexer := NewTHORChainIndexer(nil)

	testTx := []byte{0x0a, 0x01, 0x02, 0x03}
	sigs := map[string]tss.KeysignResponse{}
	pubKey := make([]byte, 33)

	_, err := indexer.ComputeTxHash(testTx, sigs, pubKey)
	if err == nil {
		t.Error("Expected error for nil SDK")
	}
}
