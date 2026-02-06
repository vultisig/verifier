package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type THORChain struct {
	baseURL    string
	httpClient *http.Client
}

func NewTHORChain(rpcURL string) (*THORChain, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Test connectivity
	resp, err := client.Get(rpcURL + "/cosmos/base/tendermint/v1beta1/node_info")
	if err != nil {
		return nil, fmt.Errorf("failed to connect to THORChain node: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("THORChain node returned status %d", resp.StatusCode)
	}

	return &THORChain{
		baseURL:    rpcURL,
		httpClient: client,
	}, nil
}

type thorchainTxResponse struct {
	TxResponse struct {
		Code   int    `json:"code"`
		TxHash string `json:"txhash"`
		Height string `json:"height"`
		RawLog string `json:"raw_log"`
	} `json:"tx_response"`
}

func (t *THORChain) GetTxStatus(ctx context.Context, txHash string) (TxStatusResult, error) {
	if ctx.Err() != nil {
		return TxStatusResult{}, ctx.Err()
	}

	url := fmt.Sprintf("%s/cosmos/tx/v1beta1/txs/%s", t.baseURL, txHash)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return TxStatusResult{}, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return TxStatusResult{}, fmt.Errorf("failed to query THORChain tx status: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return NewTxStatusResult(TxOnChainPending, ""), nil
	}

	if resp.StatusCode != http.StatusOK {
		return TxStatusResult{}, fmt.Errorf("THORChain returned unexpected status %d for tx %s", resp.StatusCode, txHash)
	}

	var txResp thorchainTxResponse
	if err := json.NewDecoder(resp.Body).Decode(&txResp); err != nil {
		return TxStatusResult{}, fmt.Errorf("failed to decode THORChain tx response: %w", err)
	}

	if txResp.TxResponse.Height == "" || txResp.TxResponse.Height == "0" {
		return NewTxStatusResult(TxOnChainPending, ""), nil
	}

	if txResp.TxResponse.Code != 0 {
		return NewTxStatusResult(TxOnChainFail, txResp.TxResponse.RawLog), nil
	}

	return NewTxStatusResult(TxOnChainSuccess, ""), nil
}
