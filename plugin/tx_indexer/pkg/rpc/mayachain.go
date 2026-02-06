package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type MayaChain struct {
	baseURL    string
	httpClient *http.Client
}

func NewMayaChain(rpcURL string) (*MayaChain, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Get(rpcURL + "/cosmos/base/tendermint/v1beta1/node_info")
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MayaChain node: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("MayaChain node returned status %d", resp.StatusCode)
	}

	return &MayaChain{
		baseURL:    rpcURL,
		httpClient: client,
	}, nil
}

type mayachainTxResponse struct {
	TxResponse struct {
		Code   int    `json:"code"`
		TxHash string `json:"txhash"`
		Height string `json:"height"`
		RawLog string `json:"raw_log"`
	} `json:"tx_response"`
}

func (m *MayaChain) GetTxStatus(ctx context.Context, txHash string) (TxStatusResult, error) {
	if ctx.Err() != nil {
		return TxStatusResult{}, ctx.Err()
	}

	url := fmt.Sprintf("%s/cosmos/tx/v1beta1/txs/%s", m.baseURL, txHash)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return TxStatusResult{}, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return TxStatusResult{}, fmt.Errorf("failed to query MayaChain tx status: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return NewTxStatusResult(TxOnChainPending, ""), nil
	}

	if resp.StatusCode != http.StatusOK {
		return TxStatusResult{}, fmt.Errorf("MayaChain returned unexpected status %d for tx %s", resp.StatusCode, txHash)
	}

	var txResp mayachainTxResponse
	err = json.NewDecoder(resp.Body).Decode(&txResp)
	if err != nil {
		return TxStatusResult{}, fmt.Errorf("failed to decode MayaChain tx response: %w", err)
	}

	if txResp.TxResponse.Height == "" || txResp.TxResponse.Height == "0" {
		return NewTxStatusResult(TxOnChainPending, ""), nil
	}

	if txResp.TxResponse.Code != 0 {
		return NewTxStatusResult(TxOnChainFail, txResp.TxResponse.RawLog), nil
	}

	return NewTxStatusResult(TxOnChainSuccess, ""), nil
}
