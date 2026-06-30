package api

import (
	"math/big"
	"testing"

	ecommon "github.com/ethereum/go-ethereum/common"
	etypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/stretchr/testify/require"
	"github.com/vultisig/recipes/engine"
	"github.com/vultisig/recipes/sdk/evm/codegen/erc20"
	"github.com/vultisig/vultisig-go/common"
)

// buildTestEvmTx encodes an unsigned EIP-1559 tx for use in gate tests.
// The resulting bytes are the same format the verifier receives from clients
// (base64-decoded from req.Transaction after ExtractTxBytes).
func buildTestEvmTx(to ecommon.Address, data []byte, value *big.Int) []byte {
	unsigned := struct {
		ChainID    *big.Int
		Nonce      uint64
		GasTipCap  *big.Int
		GasFeeCap  *big.Int
		Gas        uint64
		To         *ecommon.Address `rlp:"nil"`
		Value      *big.Int
		Data       []byte
		AccessList etypes.AccessList
	}{
		ChainID:    big.NewInt(1),
		Nonce:      0,
		GasTipCap:  big.NewInt(2_000_000_000),  // 2 gwei
		GasFeeCap:  big.NewInt(20_000_000_000), // 20 gwei
		Gas:        300_000,
		To:         &to,
		Value:      value,
		Data:       data,
		AccessList: nil,
	}
	payload, err := rlp.EncodeToBytes(unsigned)
	if err != nil {
		panic(err)
	}
	return append([]byte{etypes.DynamicFeeTxType}, payload...)
}

// TestFreedomPolicyGate covers the five security cases mandated by the design.
// FreedomPolicy is evaluated directly (no HTTP / Redis / DB needed) so these
// tests run without any external infrastructure.
func TestFreedomPolicyGate(t *testing.T) {
	t.Parallel()

	ngn, err := engine.NewEngine()
	require.NoError(t, err)

	usdc := ecommon.HexToAddress(usdcAddress)
	uint256Max, ok := new(big.Int).SetString(
		"115792089237316195423570985008687907853269984665640564039457584007913129639935", 10,
	)
	require.True(t, ok)

	t.Run("over-cap native transfer is denied", func(t *testing.T) {
		t.Parallel()
		// 0.1 ETH = 1e17 wei — exceeds 0.01 ETH cap.
		recipient := ecommon.HexToAddress("0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef")
		tx := buildTestEvmTx(recipient, nil, big.NewInt(1e17))
		_, err := ngn.Evaluate(FreedomPolicy, common.Ethereum, tx)
		require.Error(t, err, "0.1 ETH transfer must be denied (cap is 0.01 ETH)")
	})

	t.Run("unbounded erc20 approve is denied", func(t *testing.T) {
		t.Parallel()
		spender := ecommon.HexToAddress("0xabcdef1234567890abcdef1234567890abcdef12")
		data := erc20.NewErc20().PackApprove(spender, uint256Max)
		tx := buildTestEvmTx(usdc, data, big.NewInt(0))
		_, err := ngn.Evaluate(FreedomPolicy, common.Ethereum, tx)
		require.Error(t, err, "uint256.max approve must be denied by DENY rule")
		require.Contains(t, err.Error(), "denied")
	})

	t.Run("garbage tx bytes are rejected", func(t *testing.T) {
		t.Parallel()
		// Cannot decode → policy evaluation fails before rule matching.
		garbage := []byte("this is not a valid rlp evm transaction at all")
		_, err := ngn.Evaluate(FreedomPolicy, common.Ethereum, garbage)
		require.Error(t, err, "invalid tx bytes must be rejected")
	})

	t.Run("non-allowlisted calldata is denied", func(t *testing.T) {
		t.Parallel()
		// ERC-20 transfer targeting a contract that is NOT in the policy.
		// The transfer ALLOW rule targets USDC; a different token contract
		// will not match TARGET_TYPE_ADDRESS and must be rejected.
		randomToken := ecommon.HexToAddress("0x1234567890123456789012345678901234567890")
		data := erc20.NewErc20().PackTransfer(ecommon.HexToAddress("0xdeadbeef"), big.NewInt(1_000_000))
		tx := buildTestEvmTx(randomToken, data, big.NewInt(0))
		_, err := ngn.Evaluate(FreedomPolicy, common.Ethereum, tx)
		require.Error(t, err, "transfer to non-allowlisted token must be denied")
	})

	t.Run("valid in-cap usdc transfer passes", func(t *testing.T) {
		t.Parallel()
		// $1 USDC = 1_000_000 (6 decimals) — well within the $10 cap.
		recipient := ecommon.HexToAddress("0xdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef")
		data := erc20.NewErc20().PackTransfer(recipient, big.NewInt(1_000_000))
		tx := buildTestEvmTx(usdc, data, big.NewInt(0))
		rule, err := ngn.Evaluate(FreedomPolicy, common.Ethereum, tx)
		require.NoError(t, err, "valid $1 USDC transfer must be allowed")
		require.NotNil(t, rule)
		require.Equal(t, "allow-usdc-transfer", rule.GetId())
	})
}
