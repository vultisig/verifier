package chain

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"testing"

	"github.com/vultisig/mobile-tss-lib/tss"
	"github.com/vultisig/recipes/sdk/zcash"
)

// Test constants
const (
	testPubKeyHex    = "0279be667ef9dcbbac55a06295ce870b07029bfcdb2dce28d959f2815b16f81798"
	testPrevTxHash   = "9b1a2f3e4d5c6b7a8f9e1f0b4b4d5b2b4b8d3e0c8050b5b0e3f7650145cdabcd"
	testInputValue   = uint64(100000000) // 1 ZEC
	testOutputValue  = int64(99990000)   // 0.9999 ZEC
	mockDerSignature = "3044022012345678901234567890123456789012345678901234567890123456789012340220abcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcdefabcd"
)

// P2PKH script for testing
var testP2PKHScript = []byte{
	0x76, 0xa9, 0x14,
	0xab, 0xcd, 0xef, 0xab, 0xcd, 0xef, 0xab, 0xcd,
	0xef, 0xab, 0xcd, 0xef, 0xab, 0xcd, 0xef, 0xab,
	0xcd, 0xef, 0xab, 0xcd,
	0x88, 0xac,
}

func createTestTransactionWithMetadata(t *testing.T) []byte {
	t.Helper()

	sdk := zcash.NewSDK(nil)
	pubKey, err := hex.DecodeString(testPubKeyHex)
	if err != nil {
		t.Fatalf("Failed to decode pubkey: %v", err)
	}

	inputs := []zcash.TxInput{
		{
			TxHash:   testPrevTxHash,
			Index:    0,
			Value:    testInputValue,
			Script:   testP2PKHScript,
			Sequence: 0xffffffff,
		},
	}

	outputs := []*zcash.TxOutput{
		{
			Value:  testOutputValue,
			Script: testP2PKHScript,
		},
	}

	// Serialize unsigned tx
	rawBytes, err := sdk.SerializeUnsignedTx(inputs, outputs)
	if err != nil {
		t.Fatalf("SerializeUnsignedTx failed: %v", err)
	}

	// Calculate sig hash
	sigHash, err := sdk.CalculateSigHash(inputs, outputs, 0)
	if err != nil {
		t.Fatalf("CalculateSigHash failed: %v", err)
	}

	// Serialize with metadata
	return zcash.SerializeWithMetadata(rawBytes, [][]byte{sigHash}, pubKey)
}

func createTestSignatures(t *testing.T, data []byte) map[string]tss.KeysignResponse {
	t.Helper()

	_, _, sigHashes, err := zcash.ParseWithMetadata(data)
	if err != nil {
		t.Fatalf("ParseWithMetadata failed: %v", err)
	}

	sigs := make(map[string]tss.KeysignResponse)
	for _, sigHash := range sigHashes {
		derivedKey := zcash.DeriveKeyFromMessage(sigHash)
		sigs[derivedKey] = tss.KeysignResponse{
			DerSignature: mockDerSignature,
		}
	}

	return sigs
}

func TestNewZcashIndexer(t *testing.T) {
	indexer := NewZcashIndexer()
	if indexer == nil {
		t.Fatal("NewZcashIndexer returned nil")
	}
}

func TestZcashIndexer_ComputeTxHash(t *testing.T) {
	indexer := NewZcashIndexer()

	// Create test transaction with metadata
	data := createTestTransactionWithMetadata(t)

	// Create signatures
	sigs := createTestSignatures(t, data)

	// Compute tx hash
	txHash, err := indexer.ComputeTxHash(data, sigs, nil)
	if err != nil {
		t.Fatalf("ComputeTxHash failed: %v", err)
	}

	// Verify hash format (64 hex chars)
	if len(txHash) != 64 {
		t.Errorf("Expected tx hash length 64, got %d", len(txHash))
	}

	// Verify it's valid hex
	if _, err := hex.DecodeString(txHash); err != nil {
		t.Errorf("Tx hash is not valid hex: %v", err)
	}

	t.Logf("✓ ComputeTxHash: %s", txHash)
}

func TestZcashIndexer_ComputeTxHash_NoSignatures(t *testing.T) {
	indexer := NewZcashIndexer()

	data := createTestTransactionWithMetadata(t)

	// Empty signatures map
	_, err := indexer.ComputeTxHash(data, map[string]tss.KeysignResponse{}, nil)
	if err == nil {
		t.Error("Expected error for empty signatures")
	}
}

func TestZcashIndexer_ComputeTxHash_NoMetadata(t *testing.T) {
	indexer := NewZcashIndexer()

	// Plain tx bytes without metadata
	plainTxBytes := []byte{0x04, 0x00, 0x00, 0x80, 0x85, 0x20, 0x2f, 0x89}

	sigs := map[string]tss.KeysignResponse{
		"test-key": {DerSignature: mockDerSignature},
	}

	_, err := indexer.ComputeTxHash(plainTxBytes, sigs, nil)
	if err == nil {
		t.Error("Expected error for tx without metadata")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("no sighashes found")) {
		t.Errorf("Expected 'no sighashes found' error, got: %v", err)
	}
}

func TestZcashIndexer_ComputeTxHash_MissingSignature(t *testing.T) {
	indexer := NewZcashIndexer()

	data := createTestTransactionWithMetadata(t)

	// Wrong signature key
	sigs := map[string]tss.KeysignResponse{
		"wrong-key": {DerSignature: mockDerSignature},
	}

	_, err := indexer.ComputeTxHash(data, sigs, nil)
	if err == nil {
		t.Error("Expected error for missing signature")
	}
	if !bytes.Contains([]byte(err.Error()), []byte("missing signature")) {
		t.Errorf("Expected 'missing signature' error, got: %v", err)
	}
}

func TestZcashIndexer_ComputeTxHash_MultipleInputs(t *testing.T) {
	indexer := NewZcashIndexer()
	sdk := zcash.NewSDK(nil)
	pubKey, _ := hex.DecodeString(testPubKeyHex)

	inputs := []zcash.TxInput{
		{
			TxHash:   testPrevTxHash,
			Index:    0,
			Value:    50000000,
			Script:   testP2PKHScript,
			Sequence: 0xffffffff,
		},
		{
			TxHash:   testPrevTxHash,
			Index:    1,
			Value:    50000000,
			Script:   testP2PKHScript,
			Sequence: 0xffffffff,
		},
	}

	outputs := []*zcash.TxOutput{
		{
			Value:  99990000,
			Script: testP2PKHScript,
		},
	}

	rawBytes, err := sdk.SerializeUnsignedTx(inputs, outputs)
	if err != nil {
		t.Fatalf("SerializeUnsignedTx failed: %v", err)
	}

	// Calculate sig hashes for both inputs
	sigHashes := make([][]byte, len(inputs))
	for i := range inputs {
		sigHash, err := sdk.CalculateSigHash(inputs, outputs, i)
		if err != nil {
			t.Fatalf("CalculateSigHash failed for input %d: %v", i, err)
		}
		sigHashes[i] = sigHash
	}

	data := zcash.SerializeWithMetadata(rawBytes, sigHashes, pubKey)

	// Create signatures for both inputs
	sigs := make(map[string]tss.KeysignResponse)
	for _, sigHash := range sigHashes {
		derivedKey := zcash.DeriveKeyFromMessage(sigHash)
		sigs[derivedKey] = tss.KeysignResponse{
			DerSignature: mockDerSignature,
		}
	}

	txHash, err := indexer.ComputeTxHash(data, sigs, nil)
	if err != nil {
		t.Fatalf("ComputeTxHash failed: %v", err)
	}

	if len(txHash) != 64 {
		t.Errorf("Expected tx hash length 64, got %d", len(txHash))
	}

	t.Logf("✓ ComputeTxHash with multiple inputs: %s", txHash)
}

func TestZcashIndexer_ComputeTxHash_Deterministic(t *testing.T) {
	indexer := NewZcashIndexer()

	data := createTestTransactionWithMetadata(t)
	sigs := createTestSignatures(t, data)

	// Compute hash multiple times
	hash1, err := indexer.ComputeTxHash(data, sigs, nil)
	if err != nil {
		t.Fatalf("First ComputeTxHash failed: %v", err)
	}

	hash2, err := indexer.ComputeTxHash(data, sigs, nil)
	if err != nil {
		t.Fatalf("Second ComputeTxHash failed: %v", err)
	}

	// Should produce same hash
	if hash1 != hash2 {
		t.Errorf("Same input should produce same hash: %s vs %s", hash1, hash2)
	}
}

func TestZcashIndexer_ImplementsInterface(t *testing.T) {
	// Verify ZcashIndexer implements Indexer interface
	var _ Indexer = (*ZcashIndexer)(nil)
}

func TestZcashIndexer_KeyDerivationConsistency(t *testing.T) {
	// Verify that key derivation is consistent between app-recurring and verifier
	// This tests the critical signature lookup mechanism

	testSigHash := []byte("test signature hash for consistency check")

	// Method 1: Direct SDK function
	key1 := zcash.DeriveKeyFromMessage(testSigHash)

	// Method 2: Manual computation (what app-recurring does)
	hash := sha256.Sum256(testSigHash)
	key2 := base64.StdEncoding.EncodeToString(hash[:])

	if key1 != key2 {
		t.Errorf("Key derivation inconsistent: SDK=%s, manual=%s", key1, key2)
	}

	t.Logf("✓ Key derivation is consistent")
}
