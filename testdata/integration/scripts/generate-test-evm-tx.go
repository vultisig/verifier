package main

import (
	"encoding/base64"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
)

// DynamicFeeTxWithoutSignature mirrors the recipes package structure
type DynamicFeeTxWithoutSignature struct {
	ChainID    *big.Int
	Nonce      uint64
	GasTipCap  *big.Int
	GasFeeCap  *big.Int
	Gas        uint64
	To         *common.Address `rlp:"nil"`
	Value      *big.Int
	Data       []byte
	AccessList types.AccessList
}

func main() {
	chainID := big.NewInt(1) // Ethereum mainnet

	// Create a DynamicFee (EIP-1559) transaction which is the modern standard
	toAddr := common.HexToAddress("0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0")
	tx := types.NewTx(&types.DynamicFeeTx{
		ChainID:   chainID,
		Nonce:     0,
		GasTipCap: big.NewInt(1000000000),        // 1 gwei
		GasFeeCap: big.NewInt(20000000000),       // 20 gwei
		Gas:       21000,
		To:        &toAddr,
		Value:     big.NewInt(1000000000000000), // 0.001 ETH
		Data:      nil,
	})

	// Manually encode the unsigned transaction without signature fields
	// The recipes decode.go expects: [type_byte][rlp_encoded_unsigned_tx]
	unsignedTx := DynamicFeeTxWithoutSignature{
		ChainID:    chainID,
		Nonce:      0,
		GasTipCap:  big.NewInt(1000000000),
		GasFeeCap:  big.NewInt(20000000000),
		Gas:        21000,
		To:         &toAddr,
		Value:      big.NewInt(1000000000000000),
		Data:       nil,
		AccessList: types.AccessList{},
	}

	// RLP encode the unsigned transaction
	txBytes, err := rlp.EncodeToBytes(unsignedTx)
	if err != nil {
		panic(err)
	}

	// Prepend the transaction type byte (2 for DynamicFeeTx)
	typedTxBytes := append([]byte{byte(types.DynamicFeeTxType)}, txBytes...)
	txB64 := base64.StdEncoding.EncodeToString(typedTxBytes)

	// Hash-to-sign must match what verifier computes from tx bytes
	signer := types.LatestSignerForChainID(chainID)
	hash := signer.Hash(tx) // 32 bytes
	msgB64 := base64.StdEncoding.EncodeToString(hash.Bytes())

	fmt.Printf("TX_B64=%s\n", txB64)
	fmt.Printf("MSG_B64=%s\n", msgB64)
}
