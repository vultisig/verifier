package bitcoin

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/vultisig/mobile-tss-lib/tss"
	"github.com/vultisig/verifier/internal/types"
)

type Tss struct{}

func NewTss() *Tss {
	return &Tss{}
}

func (t *Tss) ComputeTxHash(proposedTxHex string, sigs []tss.KeysignResponse) (string, error) {
	return "", types.ErrChainNotImplemented

	//// TODO Draft: implement BTC tx propose for plugins first, to correctly decode it here (maybe consider PSBT)
	txBytes, err := hex.DecodeString(proposedTxHex)
	if err != nil {
		return "", fmt.Errorf("hex.DecodeString: %w", err)
	}

	var tx wire.MsgTx
	err = tx.Deserialize(bytes.NewReader(txBytes))
	if err != nil {
		return "", fmt.Errorf("tx.Deserialize: %w", err)
	}

	if len(tx.TxIn) != len(sigs) {
		return "", fmt.Errorf("input count (%d) does not match sigs count (%d)", len(tx.TxIn), len(sigs))
	}

	witness := tx.HasWitness()

	for i, in := range tx.TxIn {
		selectedSig := sigs[i]
		r, er := hex.DecodeString(selectedSig.R)
		if er != nil {
			return "", fmt.Errorf("hex.DecodeString(selectedSig.R): %w", er)
		}
		s, er := hex.DecodeString(selectedSig.S)
		if er != nil {
			return "", fmt.Errorf("hex.DecodeString(selectedSig.S): %w", er)
		}

		var sig []byte
		sig = append(sig, r...)
		sig = append(sig, s...)
		sig = append(sig, byte(txscript.SigHashAll))

		if witness {
			witnessPubKey := in.Witness[1] // must be set by tx proposer
			in.Witness = wire.TxWitness{sig, witnessPubKey}
			in.SignatureScript = nil
		} else {
			scriptSig, er2 := txscript.NewScriptBuilder().AddData(sig).Script()
			if er2 != nil {
				return "", fmt.Errorf("txscript.NewScriptBuilder: %w", er2)
			}
			in.SignatureScript = scriptSig
			in.Witness = nil
		}
	}

	return tx.TxHash().String(), nil
}
