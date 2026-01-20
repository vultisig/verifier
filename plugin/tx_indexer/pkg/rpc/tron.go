package rpc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const tronDefaultTimeout = 30 * time.Second

type Tron struct {
	rpcURL string
	client *http.Client
}

// TronTransactionInfoResponse represents the response from gettransactioninfobyid
type TronTransactionInfoResponse struct {
	ID             string      `json:"id"`
	BlockNumber    int64       `json:"blockNumber"`
	BlockTimeStamp int64       `json:"blockTimeStamp"`
	ContractResult []string    `json:"contractResult"`
	Receipt        TronReceipt `json:"receipt"`
	Result         string      `json:"result,omitempty"` // "FAILED" if failed
}

type TronReceipt struct {
	Result           string `json:"result,omitempty"` // "SUCCESS" or empty for success, "REVERT" or others for failure
	NetFee           int64  `json:"net_fee,omitempty"`
	NetUsage         int64  `json:"net_usage,omitempty"`
	EnergyUsage      int64  `json:"energy_usage,omitempty"`
	EnergyUsageTotal int64  `json:"energy_usage_total,omitempty"`
}

type TronRequest struct {
	Value string `json:"value"`
}

func NewTron(rpcURL string) (*Tron, error) {
	client := &http.Client{
		Timeout: tronDefaultTimeout,
	}

	tron := &Tron{
		rpcURL: rpcURL,
		client: client,
	}

	// Test connection by making a simple request to get node info
	testReq, err := http.NewRequest("POST", rpcURL+"/wallet/getnodeinfo", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create test request: %w", err)
	}
	testReq.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(testReq)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Tron RPC: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Tron RPC returned status %d", resp.StatusCode)
	}

	return tron, nil
}

func (t *Tron) GetTxStatus(ctx context.Context, txHash string) (TxOnChainStatus, error) {
	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	req := TronRequest{
		Value: txHash,
	}

	respBody, err := t.makeRequest(ctx, "/wallet/gettransactioninfobyid", req)
	if err != nil {
		return "", fmt.Errorf("failed to get transaction info: %w", err)
	}

	// Empty response means transaction not found (pending or doesn't exist)
	if len(respBody) == 0 || string(respBody) == "{}" {
		return TxOnChainPending, nil
	}

	var txInfo TronTransactionInfoResponse
	if err := json.Unmarshal(respBody, &txInfo); err != nil {
		return "", fmt.Errorf("failed to unmarshal Tron transaction response: %w", err)
	}

	// If ID is empty, transaction not found
	if txInfo.ID == "" {
		return TxOnChainPending, nil
	}

	// Check for explicit failure in Result field
	if txInfo.Result == "FAILED" {
		return TxOnChainFail, nil
	}

	// Check receipt result - empty or "SUCCESS" means success
	switch txInfo.Receipt.Result {
	case "", "SUCCESS":
		return TxOnChainSuccess, nil
	case "REVERT", "OUT_OF_ENERGY", "OUT_OF_TIME", "UNKNOWN":
		return TxOnChainFail, nil
	default:
		// Any other receipt result is considered a failure
		return TxOnChainFail, nil
	}
}

func (t *Tron) makeRequest(ctx context.Context, endpoint string, req interface{}) ([]byte, error) {
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", t.rpcURL+endpoint, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := t.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to make HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Tron RPC returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return body, nil
}
