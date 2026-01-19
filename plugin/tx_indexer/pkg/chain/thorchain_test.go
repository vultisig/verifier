package chain

import (
	"testing"

	"github.com/vultisig/mobile-tss-lib/tss"
	cosmossdk "github.com/vultisig/recipes/sdk/cosmos"
)

func TestTHORChainIndexer_ComputeTxHash(t *testing.T) {
	sdk := cosmossdk.NewSDK(nil)
	indexer := NewTHORChainIndexer(sdk)

	testTx := []byte{0x0a, 0x01, 0x02, 0x03}
	sigs := map[string]tss.KeysignResponse{}

	hash, err := indexer.ComputeTxHash(testTx, sigs)
	if err != nil {
		t.Fatalf("ComputeTxHash failed: %v", err)
	}

	if hash == "" {
		t.Error("Expected non-empty hash")
	}

	expectedLen := 64
	if len(hash) != expectedLen {
		t.Errorf("Expected hash length %d, got %d", expectedLen, len(hash))
	}

	t.Logf("TX Hash: %s", hash)
}

func TestTHORChainIndexer_NilSDK(t *testing.T) {
	indexer := NewTHORChainIndexer(nil)

	testTx := []byte{0x0a, 0x01, 0x02, 0x03}
	sigs := map[string]tss.KeysignResponse{}

	_, err := indexer.ComputeTxHash(testTx, sigs)
	if err == nil {
		t.Error("Expected error for nil SDK")
	}
}
