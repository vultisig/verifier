package chain

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"

	"cosmossdk.io/math"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
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
//     rely on. We assert this directly via `UnpackAny` against an Any whose
//     cachedValue has been cleared by a marshal+unmarshal roundtrip — that
//     forces the registry resolution path rather than the cached fast-path.
//
//  2. The end-to-end indexer hash path (`Sign + ComputeTxHash`) accepts the
//     new tx shapes and produces SHA256 of the signed bytes. Sign() itself
//     does not UnpackAny the body messages — it round-trips the outer tx.Tx
//     envelope — so the hash tests don't *require* the new interface
//     registrations to pass. They guard the broader contract: an indexer fed
//     an unsigned redelegate/withdraw_rewards tx (or a multi-msg body mixing
//     bank + staking, or a thor MsgDeposit + cosmos staking msg) should not
//     error, and the hash equals SHA256 of the marshalled signed bytes.
//
// If a future recipes bump regresses either invariant, the failure surfaces
// here rather than at runtime when a real redelegate broadcasts.

// Real-shape test fixtures.
//
// Addresses are real bech32 strings on cosmoshub-4. The delegator address
// matches QA-Stability-Vault's cosmoshub address; the validator addresses are
// well-known mainnet validators (Coinbase Custody, Binance Staking). These
// addresses don't need to be live for tests — the SDK's Sign()/ComputeTxHash
// path doesn't validate them — but using real shapes means a future bech32
// stricter check upstream wouldn't fail in a confusing way.
const (
	testDelegator    = "cosmos1n9qcqp4ds6c30zmnyxwq30vfd03cyuq3gpsllv"
	testValidatorSrc = "cosmosvaloper1grgelyng2v6v3t8z87wu3sxgt9m5s03xfytvz7"
	testValidatorDst = "cosmosvaloper1sjllsnramtg7ewxqwwrwjxfgc4n4ef9u2lcnj0"
	testValidator    = "cosmosvaloper1grgelyng2v6v3t8z87wu3sxgt9m5s03xfytvz7"
	testRecipient    = "cosmos1ej2es5fjztqjcd4pwa0zyvaevtjd2y5w37wr9t"
)

// testKeypair returns a deterministic secp256k1 keypair derived from a fixed
// seed. The pubkey bytes are a real on-curve compressed (33-byte) secp256k1
// point — not the off-curve `[0x02 || 32 zero-bytes]` placeholder we used in
// the first cut. Sign() in recipes doesn't validate curve membership today,
// but a stricter upstream check would reject the placeholder, so use a real
// point.
func testKeypair(t *testing.T) (*secp256k1.PrivKey, []byte) {
	t.Helper()
	priv := secp256k1.GenPrivKeyFromSecret([]byte("vultisig-verifier-cosmos-msg-types-test"))
	pub := priv.PubKey().Bytes()
	require.Len(t, pub, 33, "compressed secp256k1 pubkey must be 33 bytes")
	require.True(t, pub[0] == 0x02 || pub[0] == 0x03, "valid compressed pubkey starts with 0x02 or 0x03")
	return priv, pub
}

// testSig returns a real ECDSA signature over a deterministic message, split
// into R || S hex strings as the recipes SDK expects on the wire from a TSS
// keysign. The signature is in lower-S form (the secp256k1 Sign helper
// normalizes it). This replaces the earlier 32-byte-zero R placeholder, which
// would never come out of a real keysign.
func testSig(t *testing.T, priv *secp256k1.PrivKey) tss.KeysignResponse {
	t.Helper()
	msg := []byte("vultisig-verifier-cosmos-msg-types-test-message")
	sig, err := priv.Sign(msg)
	require.NoError(t, err)
	require.Len(t, sig, 64, "secp256k1 signature is R||S = 64 bytes")
	return tss.KeysignResponse{
		R: hex.EncodeToString(sig[:32]),
		S: hex.EncodeToString(sig[32:]),
	}
}

// buildUnsignedCosmosTx wraps any sdk.Msg into a minimal tx.Tx with no
// SignerInfos (Sign() injects the signer info from the provided pubkey).
// This mirrors the shape that vultisig-app's cosmosTx.ts emits before
// keysign — the unsigned bytes the verifier indexer receives.
func buildUnsignedCosmosTx(t *testing.T, msgs ...cosmostypes.Msg) []byte {
	t.Helper()

	anys := make([]*codectypes.Any, 0, len(msgs))
	for _, m := range msgs {
		any, err := codectypes.NewAnyWithValue(m)
		require.NoError(t, err)
		anys = append(anys, any)
	}

	body := &tx.TxBody{
		Messages: anys,
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

// expectedSignedHash signs the unsigned tx and returns the upper-hex SHA256
// of the marshalled signed bytes — i.e. the exact value ComputeTxHash should
// return. Callers use this to assert byte-for-byte equality with the indexer.
func expectedSignedHash(t *testing.T, sdk *cosmossdk.SDK, unsigned []byte, sigs map[string]tss.KeysignResponse, pubKey []byte) string {
	t.Helper()
	signed, err := sdk.Sign(unsigned, sigs, pubKey)
	require.NoError(t, err)
	sum := sha256.Sum256(signed)
	return strings.ToUpper(hex.EncodeToString(sum[:]))
}

// roundTripAny clears the cachedValue on an Any by marshalling and
// unmarshalling, forcing UnpackAny to actually consult the InterfaceRegistry
// rather than short-circuiting on the cached pointer. Without this, the
// registry-registration assertion is a no-op: NewAnyWithValue stashes the
// original *MsgBeginRedelegate on the Any, and UnpackAny just type-asserts
// against the cached value — passing even if the registry doesn't know the
// type URL at all.
//
// Reference: cosmos-sdk codec/types/interface_registry.go:UnpackAny — the
// `cachedValue != nil` branch returns before any registry lookup.
func roundTripAny(t *testing.T, original *codectypes.Any) *codectypes.Any {
	t.Helper()
	bz, err := original.Marshal()
	require.NoError(t, err)
	out := &codectypes.Any{}
	require.NoError(t, out.Unmarshal(bz))
	return out
}

// TestRecipesSDKRegistersStakingDistributionInterfaces asserts the actual
// behavior change recipes Phase 1.5 introduces: a freshly-built recipes
// Cosmos SDK can decode a /cosmos.staking.v1beta1.MsgBeginRedelegate and a
// /cosmos.distribution.v1beta1.MsgWithdrawDelegatorReward off the wire. This
// is the load-bearing assertion for any verifier consumer that goes beyond
// envelope round-tripping (e.g. policy evaluation, parameter extraction).
//
// We force the registry path by marshal+unmarshalling the Any first — see
// roundTripAny — so the cachedValue fast-path can't mask a missing
// registration. If you remove `stakingtypes.RegisterInterfaces(ir)` or
// `distributiontypes.RegisterInterfaces(ir)` from
// recipes/sdk/cosmos/sdk.go's NewSDK, this test fails with
// "no registered implementations of type ... for type URL ...".
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

		// Force the registry path; without this, UnpackAny would short-circuit
		// on the cached *MsgBeginRedelegate that NewAnyWithValue stashed.
		any = roundTripAny(t, any)

		var decoded cosmostypes.Msg
		err = ir.UnpackAny(any, &decoded)
		require.NoError(t, err, "recipes SDK must register stakingtypes.RegisterInterfaces")
		got, ok := decoded.(*stakingtypes.MsgBeginRedelegate)
		require.True(t, ok)
		assert.Equal(t, original.DelegatorAddress, got.DelegatorAddress)
		assert.Equal(t, original.ValidatorSrcAddress, got.ValidatorSrcAddress)
		assert.Equal(t, original.ValidatorDstAddress, got.ValidatorDstAddress)
		assert.Equal(t, original.Amount, got.Amount)
	})

	t.Run("MsgWithdrawDelegatorReward decodes via SDK registry", func(t *testing.T) {
		original := &distributiontypes.MsgWithdrawDelegatorReward{
			DelegatorAddress: testDelegator,
			ValidatorAddress: testValidator,
		}
		any, err := codectypes.NewAnyWithValue(original)
		require.NoError(t, err)
		assert.Equal(t, "/cosmos.distribution.v1beta1.MsgWithdrawDelegatorReward", any.TypeUrl)

		any = roundTripAny(t, any)

		var decoded cosmostypes.Msg
		err = ir.UnpackAny(any, &decoded)
		require.NoError(t, err, "recipes SDK must register distributiontypes.RegisterInterfaces")
		got, ok := decoded.(*distributiontypes.MsgWithdrawDelegatorReward)
		require.True(t, ok)
		assert.Equal(t, original.DelegatorAddress, got.DelegatorAddress)
		assert.Equal(t, original.ValidatorAddress, got.ValidatorAddress)
	})
}

// thorChainTestSDK builds an SDK wired the way chains_list.go wires the
// THORChain indexer: NewSDK + thor MsgDeposit registration + RefreshCodec.
// Tests that exercise THORChainIndexer use this so the codec sees both the
// native cosmos staking/distribution types (registered by NewSDK) and
// thor-specific ones (registered explicitly).
func thorChainTestSDK() *cosmossdk.SDK {
	sdk := cosmossdk.NewSDK(nil)
	rtypes.RegisterInterfaces(sdk.InterfaceRegistry())
	sdk.RefreshCodec()
	return sdk
}

// mayaChainTestSDK mirrors thorChainTestSDK for the Maya indexer.
// chains_list.go wires Maya with the same rtypes.RegisterInterfaces call, so
// the test stack is identical — but the indexer struct and chain ID differ,
// so we keep a distinct helper for clarity.
func mayaChainTestSDK() *cosmossdk.SDK {
	sdk := cosmossdk.NewSDK(nil)
	rtypes.RegisterInterfaces(sdk.InterfaceRegistry())
	sdk.RefreshCodec()
	return sdk
}

func TestTHORChainIndexer_HashesMsgBeginRedelegate(t *testing.T) {
	sdk := thorChainTestSDK()
	indexer := NewTHORChainIndexer(sdk)
	_, pubKey := testKeypair(t)
	priv, _ := testKeypair(t)
	sigs := map[string]tss.KeysignResponse{"signer-0": testSig(t, priv)}

	msg := &stakingtypes.MsgBeginRedelegate{
		DelegatorAddress:    testDelegator,
		ValidatorSrcAddress: testValidatorSrc,
		ValidatorDstAddress: testValidatorDst,
		Amount:              cosmostypes.NewCoin("uatom", math.NewInt(1_000_000)),
	}
	unsigned := buildUnsignedCosmosTx(t, msg)

	hash, err := indexer.ComputeTxHash(unsigned, sigs, pubKey)
	require.NoError(t, err, "indexer must accept MsgBeginRedelegate after recipes Phase 1.5")
	assert.Len(t, hash, 64, "cosmos tx hash is uppercase hex SHA256 of signed bytes")

	// Determinism: two calls with the same inputs produce the same hash.
	hash2, err := indexer.ComputeTxHash(unsigned, sigs, pubKey)
	require.NoError(t, err)
	assert.Equal(t, hash, hash2)

	// Byte-equality with SHA256(signed): the indexer's contract is exactly
	// SHA256 of the marshalled signed tx bytes — this is what
	// TestTHORChainIndexer_HashIsSignedShape used to assert via a weaker
	// not-equal-to-unsigned guard.
	assert.Equal(t, expectedSignedHash(t, sdk, unsigned, sigs, pubKey), hash)
}

func TestTHORChainIndexer_HashesMsgWithdrawDelegatorReward(t *testing.T) {
	sdk := thorChainTestSDK()
	indexer := NewTHORChainIndexer(sdk)
	_, pubKey := testKeypair(t)
	priv, _ := testKeypair(t)
	sigs := map[string]tss.KeysignResponse{"signer-0": testSig(t, priv)}

	msg := &distributiontypes.MsgWithdrawDelegatorReward{
		DelegatorAddress: testDelegator,
		ValidatorAddress: testValidator,
	}
	unsigned := buildUnsignedCosmosTx(t, msg)

	hash, err := indexer.ComputeTxHash(unsigned, sigs, pubKey)
	require.NoError(t, err, "indexer must accept MsgWithdrawDelegatorReward after recipes Phase 1.5")
	assert.Len(t, hash, 64)

	_, err = hex.DecodeString(hash)
	require.NoError(t, err, "indexer hash must be valid hex")

	assert.Equal(t, expectedSignedHash(t, sdk, unsigned, sigs, pubKey), hash)
}

// TestTHORChainIndexer_HashIsSignedTxSHA256 replaces the earlier
// HashIsSignedShape test, which only asserted that the hash differed from
// SHA256(unsigned) — a tripwire that fires on any byte-changing transform,
// not specifically on returning-the-wrong-bytes regressions. The real
// contract is: ComputeTxHash returns upper-hex SHA256 of the marshalled
// signed-tx bytes. We assert that exact equality here.
func TestTHORChainIndexer_HashIsSignedTxSHA256(t *testing.T) {
	sdk := thorChainTestSDK()
	indexer := NewTHORChainIndexer(sdk)
	_, pubKey := testKeypair(t)
	priv, _ := testKeypair(t)
	sigs := map[string]tss.KeysignResponse{"signer-0": testSig(t, priv)}

	msg := &stakingtypes.MsgBeginRedelegate{
		DelegatorAddress:    testDelegator,
		ValidatorSrcAddress: testValidatorSrc,
		ValidatorDstAddress: testValidatorDst,
		Amount:              cosmostypes.NewCoin("uatom", math.NewInt(2_000_000)),
	}
	unsigned := buildUnsignedCosmosTx(t, msg)

	hash, err := indexer.ComputeTxHash(unsigned, sigs, pubKey)
	require.NoError(t, err)

	expected := expectedSignedHash(t, sdk, unsigned, sigs, pubKey)
	assert.Equal(t, expected, hash, "indexer must return SHA256(signedTx) in upper hex")

	// Cross-check: SHA256(unsignedTx) must NOT equal the indexer's hash —
	// this covers the original regression-tripwire concern (the indexer
	// must not accidentally hash the unsigned bytes).
	unsignedSum := sha256.Sum256(unsigned)
	assert.NotEqual(t, strings.ToUpper(hex.EncodeToString(unsignedSum[:])), hash)
}

// TestMayaChainIndexerRegistersStakingDistributionInterfaces mirrors the
// THORChain wiring asserted above for the Maya indexer. chains_list.go wires
// Maya with the same NewSDK + rtypes.RegisterInterfaces call THORChain uses,
// so the new staking/distribution registrations should also be reachable
// through MayaChainIndexer's SDK. Without this guard, a future divergence
// between the two indexers' wiring (e.g. a Maya-specific SDK constructor that
// skips RefreshCodec) would silently regress redelegate/withdraw_rewards
// support on Maya only.
func TestMayaChainIndexerRegistersStakingDistributionInterfaces(t *testing.T) {
	sdk := mayaChainTestSDK()
	indexer := NewMayaChainIndexer(sdk)
	_, pubKey := testKeypair(t)
	priv, _ := testKeypair(t)
	sigs := map[string]tss.KeysignResponse{"signer-0": testSig(t, priv)}

	t.Run("MsgBeginRedelegate hashes through Maya indexer", func(t *testing.T) {
		msg := &stakingtypes.MsgBeginRedelegate{
			DelegatorAddress:    testDelegator,
			ValidatorSrcAddress: testValidatorSrc,
			ValidatorDstAddress: testValidatorDst,
			Amount:              cosmostypes.NewCoin("ucacao", math.NewInt(500_000)),
		}
		unsigned := buildUnsignedCosmosTx(t, msg)
		hash, err := indexer.ComputeTxHash(unsigned, sigs, pubKey)
		require.NoError(t, err)
		assert.Equal(t, expectedSignedHash(t, sdk, unsigned, sigs, pubKey), hash)
	})

	t.Run("MsgWithdrawDelegatorReward hashes through Maya indexer", func(t *testing.T) {
		msg := &distributiontypes.MsgWithdrawDelegatorReward{
			DelegatorAddress: testDelegator,
			ValidatorAddress: testValidator,
		}
		unsigned := buildUnsignedCosmosTx(t, msg)
		hash, err := indexer.ComputeTxHash(unsigned, sigs, pubKey)
		require.NoError(t, err)
		assert.Equal(t, expectedSignedHash(t, sdk, unsigned, sigs, pubKey), hash)
	})

	t.Run("registry decodes both msg types", func(t *testing.T) {
		ir := sdk.InterfaceRegistry()

		redelegate := &stakingtypes.MsgBeginRedelegate{
			DelegatorAddress:    testDelegator,
			ValidatorSrcAddress: testValidatorSrc,
			ValidatorDstAddress: testValidatorDst,
			Amount:              cosmostypes.NewCoin("ucacao", math.NewInt(1)),
		}
		anyR, err := codectypes.NewAnyWithValue(redelegate)
		require.NoError(t, err)
		anyR = roundTripAny(t, anyR)
		var decodedR cosmostypes.Msg
		require.NoError(t, ir.UnpackAny(anyR, &decodedR))
		_, ok := decodedR.(*stakingtypes.MsgBeginRedelegate)
		assert.True(t, ok)

		withdraw := &distributiontypes.MsgWithdrawDelegatorReward{
			DelegatorAddress: testDelegator,
			ValidatorAddress: testValidator,
		}
		anyW, err := codectypes.NewAnyWithValue(withdraw)
		require.NoError(t, err)
		anyW = roundTripAny(t, anyW)
		var decodedW cosmostypes.Msg
		require.NoError(t, ir.UnpackAny(anyW, &decodedW))
		_, ok = decodedW.(*distributiontypes.MsgWithdrawDelegatorReward)
		assert.True(t, ok)
	})
}

// TestTHORChainIndexerDecodesMultiMsgWithStakingMsg covers a tx body that
// mixes a bank.MsgSend with a staking.MsgBeginRedelegate. Real cosmos txs
// often bundle messages — e.g. a wallet that auto-claims rewards before
// delegating — and the indexer's hash path must handle the multi-msg shape
// the same way it handles single-msg.
func TestTHORChainIndexerDecodesMultiMsgWithStakingMsg(t *testing.T) {
	sdk := thorChainTestSDK()
	indexer := NewTHORChainIndexer(sdk)
	_, pubKey := testKeypair(t)
	priv, _ := testKeypair(t)
	sigs := map[string]tss.KeysignResponse{"signer-0": testSig(t, priv)}

	send := &banktypes.MsgSend{
		FromAddress: testDelegator,
		ToAddress:   testRecipient,
		Amount:      cosmostypes.NewCoins(cosmostypes.NewCoin("uatom", math.NewInt(100_000))),
	}
	redelegate := &stakingtypes.MsgBeginRedelegate{
		DelegatorAddress:    testDelegator,
		ValidatorSrcAddress: testValidatorSrc,
		ValidatorDstAddress: testValidatorDst,
		Amount:              cosmostypes.NewCoin("uatom", math.NewInt(1_000_000)),
	}

	unsigned := buildUnsignedCosmosTx(t, send, redelegate)

	hash, err := indexer.ComputeTxHash(unsigned, sigs, pubKey)
	require.NoError(t, err, "indexer must hash multi-msg bodies (bank.MsgSend + staking.MsgBeginRedelegate)")
	assert.Len(t, hash, 64)
	assert.Equal(t, expectedSignedHash(t, sdk, unsigned, sigs, pubKey), hash)

	// Also confirm the registry can decode both messages back from the
	// envelope — this is the consumer's path when inspecting bundled txs.
	var unsignedTx tx.Tx
	require.NoError(t, unsignedTx.Unmarshal(unsigned))
	require.Len(t, unsignedTx.Body.Messages, 2)

	ir := sdk.InterfaceRegistry()
	var m0, m1 cosmostypes.Msg
	require.NoError(t, ir.UnpackAny(unsignedTx.Body.Messages[0], &m0))
	require.NoError(t, ir.UnpackAny(unsignedTx.Body.Messages[1], &m1))
	_, ok0 := m0.(*banktypes.MsgSend)
	_, ok1 := m1.(*stakingtypes.MsgBeginRedelegate)
	assert.True(t, ok0, "first message must decode as bank.MsgSend")
	assert.True(t, ok1, "second message must decode as staking.MsgBeginRedelegate")
}

// TestTHORChainIndexerDecodesCrossEcosystemMsgs is the canary gomes asked
// for: a tx body that mixes THORChain's MsgDeposit (registered via
// rtypes.RegisterInterfaces) with a generic cosmos staking MsgBeginRedelegate
// (registered by NewSDK). Both must round-trip through the same THORChain
// SDK without one ecosystem's registration shadowing the other's.
//
// This isn't a tx shape that thornode would broadcast in practice, but it's
// the strongest assertion that the two registrations coexist in the same
// codec: if a future recipes change accidentally drops or replaces one set
// of registrations during NewSDK setup, this test catches it.
func TestTHORChainIndexerDecodesCrossEcosystemMsgs(t *testing.T) {
	sdk := thorChainTestSDK()
	indexer := NewTHORChainIndexer(sdk)
	_, pubKey := testKeypair(t)
	priv, _ := testKeypair(t)
	sigs := map[string]tss.KeysignResponse{"signer-0": testSig(t, priv)}

	// THORChain MsgDeposit — fields per recipes/types/thorchain.pb.go.
	deposit := &rtypes.MsgDeposit{
		Memo:   "=:ETH.ETH:0xdeadbeef",
		Signer: []byte("thor1signerxxxxxxxxxxxxxxxxxxxxxxxx"),
	}

	// Generic cosmos staking MsgBeginRedelegate.
	redelegate := &stakingtypes.MsgBeginRedelegate{
		DelegatorAddress:    testDelegator,
		ValidatorSrcAddress: testValidatorSrc,
		ValidatorDstAddress: testValidatorDst,
		Amount:              cosmostypes.NewCoin("uatom", math.NewInt(1_000_000)),
	}

	unsigned := buildUnsignedCosmosTx(t, deposit, redelegate)

	hash, err := indexer.ComputeTxHash(unsigned, sigs, pubKey)
	require.NoError(t, err, "indexer must hash bodies mixing thor MsgDeposit + cosmos staking msg")
	assert.Len(t, hash, 64)
	assert.Equal(t, expectedSignedHash(t, sdk, unsigned, sigs, pubKey), hash)

	// Both messages decode through the same registry.
	var unsignedTx tx.Tx
	require.NoError(t, unsignedTx.Unmarshal(unsigned))
	require.Len(t, unsignedTx.Body.Messages, 2)

	ir := sdk.InterfaceRegistry()
	var m0, m1 cosmostypes.Msg
	require.NoError(t, ir.UnpackAny(unsignedTx.Body.Messages[0], &m0))
	require.NoError(t, ir.UnpackAny(unsignedTx.Body.Messages[1], &m1))
	_, ok0 := m0.(*rtypes.MsgDeposit)
	_, ok1 := m1.(*stakingtypes.MsgBeginRedelegate)
	assert.True(t, ok0, "first message must decode as thorchain MsgDeposit (rtypes registration intact)")
	assert.True(t, ok1, "second message must decode as staking MsgBeginRedelegate (NewSDK registration intact)")
}
