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
	} `json:"tx_response"`
}

func (t *THORChain) GetTxStatus(ctx context.Context, txHash string) (TxOnChainStatus, error) {
	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	url := fmt.Sprintf("%s/cosmos/tx/v1beta1/txs/%s", t.baseURL, txHash)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return TxOnChainPending, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return TxOnChainPending, nil
	}

	if resp.StatusCode != http.StatusOK {
		return TxOnChainPending, nil
	}

	var txResp thorchainTxResponse
	err = json.NewDecoder(resp.Body).Decode(&txResp)
	if err != nil {
		return TxOnChainPending, nil
	}

	if txResp.TxResponse.Height == "" || txResp.TxResponse.Height == "0" {
		return TxOnChainPending, nil
	}

	if txResp.TxResponse.Code != 0 {
		return TxOnChainFail, nil
	}

	return TxOnChainSuccess, nil
}
