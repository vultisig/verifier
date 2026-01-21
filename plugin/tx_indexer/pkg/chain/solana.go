package chain

import (
	"fmt"

	"github.com/gagliardetto/binary"
	"github.com/gagliardetto/solana-go"
	"github.com/vultisig/mobile-tss-lib/tss"
	solanaSDK "github.com/vultisig/recipes/sdk/solana"
)

type SolanaIndexer struct {
	sdk *solanaSDK.SDK
}

func NewSolanaIndexer(sdk *solanaSDK.SDK) *SolanaIndexer {
	return &SolanaIndexer{
		sdk: sdk,
	}
}

func (s *SolanaIndexer) ComputeTxHash(proposedTx []byte, sigs map[string]tss.KeysignResponse, _ []byte) (string, error) {
	signed, err := s.sdk.Sign(proposedTx, sigs)
	if err != nil {
		return "", fmt.Errorf("failed to sign: %w", err)
	}

	tx, err := solana.TransactionFromDecoder(bin.NewBinDecoder(signed))
	if err != nil {
		return "", fmt.Errorf("failed to decode signed tx: %w", err)
	}

	if len(tx.Signatures) == 0 {
		return "", fmt.Errorf("transaction has no signatures")
	}

	return tx.Signatures[0].String(), nil
}
