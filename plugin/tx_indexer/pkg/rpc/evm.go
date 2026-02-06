package rpc

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

const defaultTimeout = 30 * time.Second

type Evm struct {
	client *ethclient.Client
}

func NewEvm(c context.Context, rpcURL string) (*Evm, error) {
	ctx, cancel := context.WithTimeout(c, defaultTimeout)
	defer cancel()

	cl, err := ethclient.DialContext(ctx, rpcURL)
	if err != nil {
		return nil, fmt.Errorf("ethclient.DialContext: %w", err)
	}

	return &Evm{
		client: cl,
	}, nil
}

func (r *Evm) GetTxStatus(ct context.Context, txHash string) (TxStatusResult, error) {
	ctx, cancel := context.WithTimeout(ct, defaultTimeout)
	defer cancel()

	hash := common.HexToHash(txHash)
	rec, err := r.client.TransactionReceipt(ctx, hash)
	if err != nil {
		if errors.Is(err, ethereum.NotFound) {
			return NewTxStatusResult(TxOnChainPending, ""), nil
		}
		return TxStatusResult{}, fmt.Errorf("r.client.TransactionReceipt: %w", err)
	}
	switch rec.Status {
	case 0:
		errorMsg := r.extractErrorMessage(ctx, hash, rec)
		return NewTxStatusResult(TxOnChainFail, errorMsg), nil
	case 1:
		return NewTxStatusResult(TxOnChainSuccess, ""), nil
	default:
		return TxStatusResult{}, errors.New("r.client.TransactionReceipt: unknown tx receipt status by hash=" + txHash)
	}
}

func (r *Evm) extractErrorMessage(ctx context.Context, hash common.Hash, rec *types.Receipt) string {
	tx, _, err := r.client.TransactionByHash(ctx, hash)
	if err != nil {
		return "transaction reverted"
	}

	if tx == nil || tx.To() == nil {
		return "transaction reverted"
	}

	if rec.GasUsed == tx.Gas() {
		return "out of gas"
	}

	revertData := r.getRevertData(ctx, tx, rec.BlockNumber)
	if len(revertData) == 0 {
		return "transaction reverted"
	}

	msg, ok := DecodeEVMRevert(revertData)
	if ok {
		return msg
	}

	return "transaction reverted"
}

func (r *Evm) getRevertData(ctx context.Context, tx *types.Transaction, blockNumber *big.Int) []byte {
	msg := ethereum.CallMsg{
		To:    tx.To(),
		Value: tx.Value(),
		Data:  tx.Data(),
	}

	sender, err := types.Sender(types.LatestSignerForChainID(tx.ChainId()), tx)
	if err == nil {
		msg.From = sender
	}

	out, err := r.client.CallContract(ctx, msg, blockNumber)
	if err == nil {
		return nil
	}
	if len(out) > 0 {
		return out
	}
	return extractRevertBytesFromError(err)
}
