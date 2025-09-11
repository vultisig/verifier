package chain

import (
	"bytes"
	"fmt"

	"github.com/btcsuite/btcd/wire"
	"github.com/vultisig/mobile-tss-lib/tss"
	"github.com/vultisig/recipes/sdk/btc"
)

type BitcoinIndexer struct {
	sdk *btc.SDK
}

func NewBitcoinIndexer(sdk *btc.SDK) *BitcoinIndexer {
	return &BitcoinIndexer{
		sdk: sdk,
	}
}

func (b *BitcoinIndexer) ComputeTxHash(proposedTx []byte, sigs map[string]tss.KeysignResponse) (string, error) {
	signed, err := b.sdk.Sign(proposedTx, sigs)
	if err != nil {
		return "", fmt.Errorf("failed to sign: %w", err)
	}

	tx := wire.NewMsgTx(wire.TxVersion)
	err = tx.Deserialize(bytes.NewReader(signed))
	if err != nil {
		return "", fmt.Errorf("failed to deserialize signed tx: %w", err)
	}
	return tx.TxID(), nil
}
