package evm

import (
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/vultisig/mobile-tss-lib/tss"
	"github.com/vultisig/recipes/ethereum"
	"math/big"
)

type Tss struct {
	chainID *big.Int
}

func NewTss(chainID int) *Tss {
	return &Tss{
		chainID: big.NewInt(int64(chainID)),
	}
}

func (t *Tss) ComputeTxHash(proposedTxHex string, sigs []tss.KeysignResponse) (string, error) {
	if len(sigs) != 1 {
		return "", fmt.Errorf("expected exactly one signature, got %d", len(sigs))
	}

	payloadDecoded, err := ethereum.DecodeUnsignedPayload(common.FromHex(proposedTxHex))
	if err != nil {
		return "", fmt.Errorf("ethereum.DecodeUnsignedPayload: %w", err)
	}

	var sig []byte
	sig = append(sig, common.FromHex(sigs[0].R)...)
	sig = append(sig, common.FromHex(sigs[0].S)...)
	sig = append(sig, common.FromHex(sigs[0].RecoveryID)...)

	tx, err := gethtypes.NewTx(payloadDecoded).WithSignature(gethtypes.NewPragueSigner(t.chainID), sig)
	if err != nil {
		return "", fmt.Errorf("gethtypes.NewTx.WithSignature: %w", err)
	}
	return tx.Hash().Hex(), nil
}
