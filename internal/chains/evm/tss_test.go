package evm

import (
	"bytes"
	gethcommon "github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/stretchr/testify/require"
	"github.com/vultisig/mobile-tss-lib/tss"
	"github.com/vultisig/recipes/ethereum"
	"github.com/vultisig/verifier/common"
	"github.com/vultisig/verifier/internal/conv"
	"math/big"
	"testing"
)

func TestTss_ComputeTxHash_Ethereum(t *testing.T) {
	// https://etherscan.io/tx/0xfb58176cf444f9887751a07f749549799b9e6e0a398faa4e29a5cd9a90fa295a
	expectedTxHash := "0xfb58176cf444f9887751a07f749549799b9e6e0a398faa4e29a5cd9a90fa295a"

	ethID, err := common.Ethereum.EVMChainID()
	require.Nil(t, err, "common.Ethereum.EVMChainID")

	buf := &bytes.Buffer{}

	err = buf.WriteByte(gethtypes.DynamicFeeTxType)
	require.Nil(t, err, "buf.WriteByte(gethtypes.DynamicFeeTxType)")

	err = rlp.Encode(buf, &ethereum.DynamicFeeTxWithoutSignature{
		ChainID:   big.NewInt(int64(ethID)),
		Nonce:     2553547,
		GasTipCap: big.NewInt(0),
		GasFeeCap: big.NewInt(5714758749),
		Gas:       23100,
		To:        conv.Ptr(gethcommon.HexToAddress("0x087b027b0573d4f01345ef8d081e0e7d3b378d14")),
		Value:     big.NewInt(25767654731246261),
	})
	require.Nil(t, err, "rlp.Encode")

	txHash, err := NewTss(ethID).ComputeTxHash(gethcommon.Bytes2Hex(buf.Bytes()), []tss.KeysignResponse{{
		R:          "d55e81731a80a10a66475fb52021b03b9173359a3220c10db76739b674355f7a",
		S:          "6ebf679597e97da64d048e28fe418b2ca0ef08c2a0583b97d11703dc11cd727b",
		RecoveryID: "01",
	}})
	require.Nil(t, err, "NewTss(ethID).ComputeTxHash")
	require.Equal(t, expectedTxHash, txHash, "NewTss(ethID).ComputeTxHash")
}
