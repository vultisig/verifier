package chain

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/vultisig/mobile-tss-lib/tss"
	"github.com/vultisig/recipes/sdk/thorchain"
)

type THORChainIndexer struct {
	sdk *thorchain.SDK
}

func NewTHORChainIndexer(sdk *thorchain.SDK) *THORChainIndexer {
	return &THORChainIndexer{
		sdk: sdk,
	}
}

func (t *THORChainIndexer) ComputeTxHash(proposedTx []byte, sigs map[string]tss.KeysignResponse) (string, error) {
	signed, err := t.sdk.Sign(proposedTx, sigs)
	if err != nil {
		return "", fmt.Errorf("failed to sign: %w", err)
	}

	// For Cosmos-based chains like THORChain, the transaction hash is computed 
	// according to CometBFT/Tendermint standards: SHA256 hash of the signed transaction bytes,
	// but this needs to match exactly what the blockchain network expects.
	// The SDK Sign() method returns the final serialized transaction that will be broadcast.
	hash := sha256.Sum256(signed)
	return strings.ToUpper(hex.EncodeToString(hash[:])), nil
}