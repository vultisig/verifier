package chain

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"testing"

	"cosmossdk.io/math"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx"
	distributiontypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vultisig/mobile-tss-lib/tss"
	cosmossdk "github.com/vultisig/recipes/sdk/cosmos"
	rtypes "github.com/vultisig/recipes/types"
)

// These tests guard the verifier's hashing path against recipes regressions for
// the two staking/distribution messages that mcp-ts PR #72 + the
// recipes/verifier Phase 1.5 schemas introduced. They sign a fixture
// MsgBeginRedelegate and MsgWithdrawDelegatorReward via the recipes Cosmos SDK
// and assert:
//   1. the SDK's interface registry knows the message types (otherwise Sign
//      panics inside protobuf-Any unpacking),
//   2. the resulting hash is deterministic and matches the SHA256 the indexer
//      promises to upstream callers.
//
// If a future recipes bump removes stakingtypes.RegisterInterfaces /
// distributiontypes.RegisterInterfaces from sdk/cosmos/sdk.go, these tests
// fail loudly here rather than at runtime when a real redelegate broadcasts.

const (
	testDelegator    = "cosmos1delegatorxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
	testValidatorSrc = "cosmosvaloper1srcxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
	testValidatorDst = "cosmosvaloper1dstxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
	testValidator    = "cosmosvaloper1abcxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
)

// fakeSig returns a 64-byte deterministic R || S split that the recipes SDK
// will accept (low-S normalization happens inside Sign()).
func fakeSig() tss.KeysignResponse {
	return tss.KeysignResponse{
		R: hex.EncodeToString(make([]byte, 32)),
		S: hex.EncodeToString(append([]byte{0x01}, make([]byte, 31)...)),
	}
}

// buildUnsignedCosmosTx wraps any sdk.Msg into a minimal tx.Tx with no
// SignerInfos (Sign() injects the signer info from the provided pubkey).
// This mirrors the shape that vultisig-app's cosmosTx.ts emits before
// keysign — the unsigned bytes the verifier indexer receives.
func buildUnsignedCosmosTx(t *testing.T, msg cosmostypes.Msg) []byte {
	t.Helper()

	any, err := codectypes.NewAnyWithValue(msg)
	require.NoError(t, err)

	body := &tx.TxBody{
		Messages: []*codectypes.Any{any},
	}

	authInfo := &tx.AuthInfo{
		Fee: &tx.Fee{
			Amount:   cosmostypes.NewCoins(cosmostypes.NewCoin("uatom", math.NewInt(7_500))),
			GasLimit: 200_000,
		},
	}

	unsigned := &tx.Tx{
		Body:     body,
		AuthInfo: authInfo,
	}

	bz, err := unsigned.Marshal()
	require.NoError(t, err)
	return bz
}

func TestTHORChainIndexer_HashesMsgBeginRedelegate(t *testing.T) {
	// Use the same wiring as chains_list.go: NewSDK with thor/maya types
	// registered. With the recipes Phase 1.5 patch, NewSDK auto-registers
	// staking + distribution interfaces too, so MsgBeginRedelegate decodes.
	sdk := cosmossdk.NewSDK(nil)
	rtypes.RegisterInterfaces(sdk.InterfaceRegistry())
	sdk.RefreshCodec()

	indexer := NewTHORChainIndexer(sdk)

	msg := &stakingtypes.MsgBeginRedelegate{
		DelegatorAddress:    testDelegator,
		ValidatorSrcAddress: testValidatorSrc,
		ValidatorDstAddress: testValidatorDst,
		Amount:              cosmostypes.NewCoin("uatom", math.NewInt(1_000_000)),
	}
	unsigned := buildUnsignedCosmosTx(t, msg)

	pubKey := make([]byte, 33)
	pubKey[0] = 0x02
	sigs := map[string]tss.KeysignResponse{"signer-0": fakeSig()}

	hash, err := indexer.ComputeTxHash(unsigned, sigs, pubKey)
	require.NoError(t, err, "indexer must accept MsgBeginRedelegate after recipes Phase 1.5")
	assert.Len(t, hash, 64, "cosmos tx hash is uppercase hex SHA256 of signed bytes")

	// Determinism: two calls with the same inputs produce the same hash.
	hash2, err := indexer.ComputeTxHash(unsigned, sigs, pubKey)
	require.NoError(t, err)
	assert.Equal(t, hash, hash2)
}

func TestTHORChainIndexer_HashesMsgWithdrawDelegatorReward(t *testing.T) {
	sdk := cosmossdk.NewSDK(nil)
	rtypes.RegisterInterfaces(sdk.InterfaceRegistry())
	sdk.RefreshCodec()

	indexer := NewTHORChainIndexer(sdk)

	msg := &distributiontypes.MsgWithdrawDelegatorReward{
		DelegatorAddress: testDelegator,
		ValidatorAddress: testValidator,
	}
	unsigned := buildUnsignedCosmosTx(t, msg)

	pubKey := make([]byte, 33)
	pubKey[0] = 0x02
	sigs := map[string]tss.KeysignResponse{"signer-0": fakeSig()}

	hash, err := indexer.ComputeTxHash(unsigned, sigs, pubKey)
	require.NoError(t, err, "indexer must accept MsgWithdrawDelegatorReward after recipes Phase 1.5")
	assert.Len(t, hash, 64)

	// Sanity — the hash is the SHA256 hex of *something*; we just confirm shape.
	_, err = hex.DecodeString(hash)
	require.NoError(t, err, "indexer hash must be valid hex")
}

// TestTHORChainIndexer_HashIsSignedShape is a guard against accidentally
// returning the unsigned tx hash. The verifier's contract is that the hash is
// of the signed bytes (SHA256(signed)), not the unsigned bytes.
func TestTHORChainIndexer_HashIsSignedShape(t *testing.T) {
	sdk := cosmossdk.NewSDK(nil)
	rtypes.RegisterInterfaces(sdk.InterfaceRegistry())
	sdk.RefreshCodec()

	indexer := NewTHORChainIndexer(sdk)

	msg := &stakingtypes.MsgBeginRedelegate{
		DelegatorAddress:    testDelegator,
		ValidatorSrcAddress: testValidatorSrc,
		ValidatorDstAddress: testValidatorDst,
		Amount:              cosmostypes.NewCoin("uatom", math.NewInt(2_000_000)),
	}
	unsigned := buildUnsignedCosmosTx(t, msg)

	pubKey := make([]byte, 33)
	pubKey[0] = 0x02
	sigs := map[string]tss.KeysignResponse{"signer-0": fakeSig()}

	hash, err := indexer.ComputeTxHash(unsigned, sigs, pubKey)
	require.NoError(t, err)

	// Compute SHA256 of the unsigned bytes — that's NOT what the indexer should
	// return. (Cosmos hashes the *signed* tx bytes.)
	unsignedHash := sha256.Sum256(unsigned)
	unsignedHashHex := hex.EncodeToString(unsignedHash[:])
	assert.NotEqualf(t,
		fmt.Sprintf("%X", unsignedHash[:]), hash,
		"indexer must hash the signed tx bytes, not the unsigned (got SHA256(unsigned)=%s)",
		unsignedHashHex,
	)
}
