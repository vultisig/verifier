package chain

import (
	"fmt"

	"github.com/vultisig/mobile-tss-lib/tss"
	"github.com/vultisig/recipes/chain/utxo/zcash"
)

type ZcashIndexer struct {
	chain *zcash.Zcash
}

func NewZcashIndexer() *ZcashIndexer {
	return &ZcashIndexer{
		chain: zcash.NewChain().(*zcash.Zcash),
	}
}

func (z *ZcashIndexer) ComputeTxHash(proposedTx []byte, sigs map[string]tss.KeysignResponse) (string, error) {
	// Convert map to slice - Zcash expects signatures in order of inputs
	// The map key is the derived key from the message hash, but for Zcash
	// we need to pass signatures in input order
	sigSlice := make([]tss.KeysignResponse, 0, len(sigs))
	for _, sig := range sigs {
		sigSlice = append(sigSlice, sig)
	}

	if len(sigSlice) == 0 {
		return "", fmt.Errorf("no signatures provided")
	}

	return z.chain.ComputeTxHash(proposedTx, sigSlice)
}

