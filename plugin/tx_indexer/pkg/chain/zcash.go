package chain

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"fmt"

	"github.com/vultisig/mobile-tss-lib/tss"
	"github.com/vultisig/recipes/chain/utxo/zcash"
	"golang.org/x/crypto/blake2b"
)

// ZcashIndexer handles Zcash transaction hash computation for the tx_indexer
type ZcashIndexer struct {
	chain *zcash.Zcash
}

// NewZcashIndexer creates a new ZcashIndexer instance
func NewZcashIndexer() *ZcashIndexer {
	chain, ok := zcash.NewChain().(*zcash.Zcash)
	if !ok {
		// This should never happen as NewChain always returns *Zcash
		panic("zcash.NewChain() did not return *zcash.Zcash")
	}
	return &ZcashIndexer{
		chain: chain,
	}
}

// ComputeTxHash computes the transaction hash for a signed Zcash transaction.
// It orders signatures correctly by calculating the signature hash for each input
// and looking up the corresponding signature from the map using the derived key.
func (z *ZcashIndexer) ComputeTxHash(proposedTx []byte, sigs map[string]tss.KeysignResponse) (string, error) {
	if len(sigs) == 0 {
		return "", fmt.Errorf("no signatures provided")
	}

	// Parse transaction to get inputs for signature hash calculation
	inputs, outputs, err := parseZcashTxForSigHash(proposedTx)
	if err != nil {
		return "", fmt.Errorf("failed to parse transaction: %w", err)
	}

	// Build signature slice in input order by calculating sighash for each input
	// and looking up the signature using the derived key
	sigSlice := make([]tss.KeysignResponse, len(inputs))
	for i := range inputs {
		sigHash, err := calculateZcashSigHash(inputs, outputs, i)
		if err != nil {
			return "", fmt.Errorf("failed to calculate sig hash for input %d: %w", i, err)
		}

		derivedKey := deriveKeyFromMessage(sigHash)
		sig, exists := sigs[derivedKey]
		if !exists {
			return "", fmt.Errorf("missing signature for input %d (derived key: %s)", i, derivedKey)
		}
		sigSlice[i] = sig
	}

	return z.chain.ComputeTxHash(proposedTx, sigSlice)
}

// zcashInput represents a parsed input for signature hash calculation
type zcashInput struct {
	txHash   []byte // 32 bytes, reversed
	index    uint32
	script   []byte
	value    uint64
	sequence uint32
}

// zcashOutput represents a parsed output for signature hash calculation
type zcashOutput struct {
	value  uint64
	script []byte
}

// parseZcashTxForSigHash parses a Zcash v4 transaction to extract inputs and outputs
// needed for signature hash calculation
func parseZcashTxForSigHash(txBytes []byte) ([]zcashInput, []zcashOutput, error) {
	r := bytes.NewReader(txBytes)

	// Read header (version with overwintered flag)
	var header uint32
	if err := binary.Read(r, binary.LittleEndian, &header); err != nil {
		return nil, nil, fmt.Errorf("failed to read header: %w", err)
	}

	// Read version group ID
	var versionGroupID uint32
	if err := binary.Read(r, binary.LittleEndian, &versionGroupID); err != nil {
		return nil, nil, fmt.Errorf("failed to read version group ID: %w", err)
	}

	// Read inputs count
	inputCount, err := readCompactSize(r)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read input count: %w", err)
	}

	inputs := make([]zcashInput, inputCount)
	for i := uint64(0); i < inputCount; i++ {
		// Read previous output hash (32 bytes)
		hash := make([]byte, 32)
		if _, err := r.Read(hash); err != nil {
			return nil, nil, fmt.Errorf("failed to read prev hash: %w", err)
		}

		// Read previous output index
		var index uint32
		if err := binary.Read(r, binary.LittleEndian, &index); err != nil {
			return nil, nil, fmt.Errorf("failed to read prev index: %w", err)
		}

		// Read signature script
		scriptLen, err := readCompactSize(r)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to read script length: %w", err)
		}
		script := make([]byte, scriptLen)
		if _, err := r.Read(script); err != nil {
			return nil, nil, fmt.Errorf("failed to read script: %w", err)
		}

		// Read sequence
		var sequence uint32
		if err := binary.Read(r, binary.LittleEndian, &sequence); err != nil {
			return nil, nil, fmt.Errorf("failed to read sequence: %w", err)
		}

		// Extract value from scriptSig if present (for pre-populated unsigned tx)
		// The scriptSig in unsigned transactions contains value info
		var value uint64
		if len(script) >= 8 {
			// Try to extract value - it may be encoded at the start
			value = binary.LittleEndian.Uint64(script[:8])
		}

		inputs[i] = zcashInput{
			txHash:   hash,
			index:    index,
			script:   script,
			value:    value,
			sequence: sequence,
		}
	}

	// Read outputs count
	outputCount, err := readCompactSize(r)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read output count: %w", err)
	}

	outputs := make([]zcashOutput, outputCount)
	for i := uint64(0); i < outputCount; i++ {
		// Read value
		var value uint64
		if err := binary.Read(r, binary.LittleEndian, &value); err != nil {
			return nil, nil, fmt.Errorf("failed to read output value: %w", err)
		}

		// Read pk script
		scriptLen, err := readCompactSize(r)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to read output script length: %w", err)
		}
		script := make([]byte, scriptLen)
		if _, err := r.Read(script); err != nil {
			return nil, nil, fmt.Errorf("failed to read output script: %w", err)
		}

		outputs[i] = zcashOutput{
			value:  value,
			script: script,
		}
	}

	return inputs, outputs, nil
}

// NU5 consensus branch ID for signature hash personalization
// Although we use v4 transactions (Sapling format), we must use the
// consensus branch ID of the current epoch (NU5/NU6) for signature hashing.
const nu5BranchID = 0xc2d6d0b4

// calculateZcashSigHash computes the signature hash for a Zcash transparent input
// using the ZIP-243 signature hash algorithm for v4 (Sapling) transactions
func calculateZcashSigHash(inputs []zcashInput, outputs []zcashOutput, inputIndex int) ([]byte, error) {
	var preimage bytes.Buffer

	// 1. nVersion | nVersionGroupId (header)
	_ = binary.Write(&preimage, binary.LittleEndian, uint32(0x80000004)) // v4 with overwintered
	_ = binary.Write(&preimage, binary.LittleEndian, uint32(0x892F2085)) // Sapling version group

	// 2. hashPrevouts - BLAKE2b-256 of all prevouts
	hashPrevouts := calcHashPrevoutsZ(inputs)
	preimage.Write(hashPrevouts)

	// 3. hashSequence - BLAKE2b-256 of all sequences
	hashSequence := calcHashSequenceZ(inputs)
	preimage.Write(hashSequence)

	// 4. hashOutputs - BLAKE2b-256 of all outputs
	hashOutputs := calcHashOutputsZ(outputs)
	preimage.Write(hashOutputs)

	// 5. hashJoinSplits - 32 zero bytes (no joinsplits)
	preimage.Write(make([]byte, 32))

	// 6. hashShieldedSpends - 32 zero bytes (no shielded spends)
	preimage.Write(make([]byte, 32))

	// 7. hashShieldedOutputs - 32 zero bytes (no shielded outputs)
	preimage.Write(make([]byte, 32))

	// 8. nLockTime
	_ = binary.Write(&preimage, binary.LittleEndian, uint32(0))

	// 9. nExpiryHeight
	_ = binary.Write(&preimage, binary.LittleEndian, uint32(0))

	// 10. valueBalance (8 bytes) - 0 for transparent-only
	_ = binary.Write(&preimage, binary.LittleEndian, int64(0))

	// 11. nHashType
	_ = binary.Write(&preimage, binary.LittleEndian, uint32(1)) // SIGHASH_ALL

	// For SIGHASH_ALL, include the input being signed
	if inputIndex >= 0 && inputIndex < len(inputs) {
		input := inputs[inputIndex]

		// prevout (txid + index) - txHash is already reversed from parsing
		preimage.Write(input.txHash)
		_ = binary.Write(&preimage, binary.LittleEndian, input.index)

		// scriptCode (with length prefix)
		writeCompactSizeZ(&preimage, uint64(len(input.script)))
		preimage.Write(input.script)

		// amount (value of the input)
		_ = binary.Write(&preimage, binary.LittleEndian, input.value)

		// nSequence
		_ = binary.Write(&preimage, binary.LittleEndian, input.sequence)
	}

	// Final hash using BLAKE2b-256 with personalization
	return blake2bSigHashZ(preimage.Bytes())
}

// blake2bSigHashZ computes BLAKE2b-256 with Zcash signature hash personalization
func blake2bSigHashZ(data []byte) ([]byte, error) {
	personalization := make([]byte, 16)
	copy(personalization, "ZcashSigHash")
	binary.LittleEndian.PutUint32(personalization[12:], nu5BranchID)

	h, err := blake2b.New256(personalization)
	if err != nil {
		return nil, fmt.Errorf("failed to create BLAKE2b hasher: %w", err)
	}
	h.Write(data)
	return h.Sum(nil), nil
}

// calcHashPrevoutsZ computes BLAKE2b-256 of all input prevouts
func calcHashPrevoutsZ(inputs []zcashInput) []byte {
	var buf bytes.Buffer
	for _, input := range inputs {
		buf.Write(input.txHash)
		_ = binary.Write(&buf, binary.LittleEndian, input.index)
	}

	personalization := make([]byte, 16)
	copy(personalization, "ZcashPrevoutHash")
	h, _ := blake2b.New256(personalization)
	h.Write(buf.Bytes())
	return h.Sum(nil)
}

// calcHashSequenceZ computes BLAKE2b-256 of all input sequences
func calcHashSequenceZ(inputs []zcashInput) []byte {
	var buf bytes.Buffer
	for _, input := range inputs {
		_ = binary.Write(&buf, binary.LittleEndian, input.sequence)
	}

	personalization := make([]byte, 16)
	copy(personalization, "ZcashSequencHash")
	h, _ := blake2b.New256(personalization)
	h.Write(buf.Bytes())
	return h.Sum(nil)
}

// calcHashOutputsZ computes BLAKE2b-256 of all outputs
func calcHashOutputsZ(outputs []zcashOutput) []byte {
	var buf bytes.Buffer
	for _, output := range outputs {
		_ = binary.Write(&buf, binary.LittleEndian, output.value)
		writeCompactSizeZ(&buf, uint64(len(output.script)))
		buf.Write(output.script)
	}

	personalization := make([]byte, 16)
	copy(personalization, "ZcashOutputsHash")
	h, _ := blake2b.New256(personalization)
	h.Write(buf.Bytes())
	return h.Sum(nil)
}

// deriveKeyFromMessage derives a map key from a message hash using SHA256 + Base64
func deriveKeyFromMessage(messageHash []byte) string {
	hash := sha256.Sum256(messageHash)
	return base64.StdEncoding.EncodeToString(hash[:])
}

// readCompactSize reads a Bitcoin-style compact size from a reader
func readCompactSize(r *bytes.Reader) (uint64, error) {
	b, err := r.ReadByte()
	if err != nil {
		return 0, err
	}

	switch {
	case b < 0xFD:
		return uint64(b), nil
	case b == 0xFD:
		var v uint16
		if err := binary.Read(r, binary.LittleEndian, &v); err != nil {
			return 0, err
		}
		return uint64(v), nil
	case b == 0xFE:
		var v uint32
		if err := binary.Read(r, binary.LittleEndian, &v); err != nil {
			return 0, err
		}
		return uint64(v), nil
	default:
		var v uint64
		if err := binary.Read(r, binary.LittleEndian, &v); err != nil {
			return 0, err
		}
		return v, nil
	}
}

// writeCompactSizeZ writes a Bitcoin-style compact size to a buffer
func writeCompactSizeZ(buf *bytes.Buffer, v uint64) {
	switch {
	case v < 0xFD:
		buf.WriteByte(byte(v))
	case v <= 0xFFFF:
		buf.WriteByte(0xFD)
		_ = binary.Write(buf, binary.LittleEndian, uint16(v))
	case v <= 0xFFFFFFFF:
		buf.WriteByte(0xFE)
		_ = binary.Write(buf, binary.LittleEndian, uint32(v))
	default:
		buf.WriteByte(0xFF)
		_ = binary.Write(buf, binary.LittleEndian, v)
	}
}
