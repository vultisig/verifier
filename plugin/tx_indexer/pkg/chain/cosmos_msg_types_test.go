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

// These tests guard two distinct invariants the verifier needs after recipes
// Phase 1.5 lands the staking/distribution schemas:
//
//  1. The recipes Cosmos SDK's InterfaceRegistry knows the new message types.
//     This is what consumers of `sdk.InterfaceRegistry()` (including downstream
//     decoders that DO need to UnpackAny the body messages — e.g. anything
//     that inspects the redelegate amount or validator addresses post-sign)
//     rely on. We assert this directly via `UnpackAny` in
//     TestRecipesSDKRegistersStakingDistributionInterfaces.
//
//  2. The end-to-end indexer hash path (`Sign + ComputeTxHash`) accepts the
//     new tx shapes and produces a stable SHA256 of the signed bytes. Note
//     that Sign() itself only round-trips the outer tx.Tx envelope — it does
//     not UnpackAny the body messages — so the hash tests don't *require*
//     the new interface registrations to pass. They guard the broader
//     contract: an indexer fed an unsigned redelegate/withdraw_rewards tx
//     should not error, and the hash is shaped like all other cosmos hashes.
//
// If a future recipes bump regresses either invariant, the failure surfaces
// here rather than at runtime when a real redelegate broadcasts.

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

// TestRecipesSDKRegistersStakingDistributionInterfaces asserts the actual
// behavior change recipes Phase 1.5 introduces: a freshly-built recipes
// Cosmos SDK can decode a /cosmos.staking.v1beta1.MsgBeginRedelegate and a
// /cosmos.distribution.v1beta1.MsgWithdrawDelegatorReward off the wire. This
// is the load-bearing assertion for any verifier consumer that goes beyond
// envelope round-tripping (e.g. policy evaluation, parameter extraction).
func TestRecipesSDKRegistersStakingDistributionInterfaces(t *testing.T) {
	sdk := cosmossdk.NewSDK(nil)
	ir := sdk.InterfaceRegistry()

	t.Run("MsgBeginRedelegate decodes via SDK registry", func(t *testing.T) {
		original := &stakingtypes.MsgBeginRedelegate{
			DelegatorAddress:    testDelegator,
			ValidatorSrcAddress: testValidatorSrc,
			ValidatorDstAddress: testValidatorDst,
			Amount:              cosmostypes.NewCoin("uatom", math.NewInt(1_000_000)),
		}
		any, err := codectypes.NewAnyWithValue(original)
		require.NoError(t, err)
		assert.Equal(t, "/cosmos.staking.v1beta1.MsgBeginRedelegate", any.TypeUrl)

		var decoded cosmostypes.Msg
		err = ir.UnpackAny(any, &decoded)
		require.NoError(t, err, "recipes SDK must register stakingtypes.RegisterInterfaces")
		_, ok := decoded.(*stakingtypes.MsgBeginRedelegate)
		assert.True(t, ok)
	})

	t.Run("MsgWithdrawDelegatorReward decodes via SDK registry", func(t *testing.T) {
		original := &distributiontypes.MsgWithdrawDelegatorReward{
			DelegatorAddress: testDelegator,
			ValidatorAddress: testValidator,
		}
		any, err := codectypes.NewAnyWithValue(original)
		require.NoError(t, err)
		assert.Equal(t, "/cosmos.distribution.v1beta1.MsgWithdrawDelegatorReward", any.TypeUrl)

		var decoded cosmostypes.Msg
		err = ir.UnpackAny(any, &decoded)
		require.NoError(t, err, "recipes SDK must register distributiontypes.RegisterInterfaces")
		_, ok := decoded.(*distributiontypes.MsgWithdrawDelegatorReward)
		assert.True(t, ok)
	})
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
