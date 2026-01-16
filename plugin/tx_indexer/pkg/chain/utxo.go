package chain

import (
	"bytes"
	"fmt"

	"github.com/btcsuite/btcd/btcutil/psbt"
	"github.com/btcsuite/btcd/wire"
	"github.com/vultisig/mobile-tss-lib/tss"
	"github.com/vultisig/recipes/sdk/bch"
	"github.com/vultisig/recipes/sdk/btc"
	"github.com/vultisig/vultisig-go/common"
)

// UtxoSDK defines the interface for UTXO chain SDKs used by the indexer.
type UtxoSDK interface {
	Sign(psbtBytes []byte, signatures map[string]tss.KeysignResponse) ([]byte, error)
	CalculateInputSignatureHash(pkt *psbt.Packet, inputIndex int) ([]byte, error)
}

// UtxoIndexer is a generic indexer for UTXO chains that use BTC-compatible wire format.
// This includes Litecoin, Dogecoin, and Bitcoin Cash.
//
// Transaction format compatibility:
// - All these chains use the same transaction serialization format as Bitcoin (btcd/wire)
// - Txid is computed identically: double SHA256 of the serialized transaction
// - The btcd/wire.MsgTx.Deserialize() correctly parses transactions from all these chains
//
// Note: Signing differences (e.g., BCH's SIGHASH_FORKID) are handled by chain-specific SDKs.
type UtxoIndexer struct {
	chain common.Chain
	sdk   UtxoSDK
}

// NewUtxoIndexer creates a new UTXO chain indexer for the specified chain.
func NewUtxoIndexer(chain common.Chain) *UtxoIndexer {
	return &UtxoIndexer{
		chain: chain,
		sdk:   btc.NewSDK(nil),
	}
}

// NewLitecoinIndexer creates a Litecoin chain indexer.
func NewLitecoinIndexer() *UtxoIndexer {
	return NewUtxoIndexer(common.Litecoin)
}

// NewDogecoinIndexer creates a Dogecoin chain indexer.
func NewDogecoinIndexer() *UtxoIndexer {
	return NewUtxoIndexer(common.Dogecoin)
}

// NewBitcoinCashIndexer creates a Bitcoin Cash chain indexer.
// BCH requires its own SDK because it uses SIGHASH_FORKID and BIP143-style signature hashing.
func NewBitcoinCashIndexer() *UtxoIndexer {
	return &UtxoIndexer{
		chain: common.BitcoinCash,
		sdk:   bch.NewSDK(nil),
	}
}

// ComputeTxHash computes the transaction hash for a signed UTXO transaction.
// It uses the BTC SDK for signing since all these chains use compatible PSBT format.
func (u *UtxoIndexer) ComputeTxHash(proposedTx []byte, sigs map[string]tss.KeysignResponse) (string, error) {
	signed, err := u.sdk.Sign(proposedTx, sigs)
	if err != nil {
		return "", fmt.Errorf("failed to sign %s tx: %w", u.chain.String(), err)
	}

	tx := wire.NewMsgTx(wire.TxVersion)
	err = tx.Deserialize(bytes.NewReader(signed))
	if err != nil {
		return "", fmt.Errorf("failed to deserialize signed %s tx: %w", u.chain.String(), err)
	}

	return tx.TxID(), nil
}
