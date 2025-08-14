package chain

import "github.com/vultisig/mobile-tss-lib/tss"

type Indexer interface {
	ComputeTxHash(proposedTx []byte, sigs []tss.KeysignResponse) (string, error)
}
