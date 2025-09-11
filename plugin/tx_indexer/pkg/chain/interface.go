package chain

import "github.com/vultisig/mobile-tss-lib/tss"

type Indexer interface {
	ComputeTxHash(proposedTx []byte, sigs map[string]tss.KeysignResponse) (string, error)
}
