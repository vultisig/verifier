package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const tronDefaultTimeout = 30 * time.Second

// Tron implements the Rpc interface for TRON chain
type Tron struct {
	baseURL string
	client  *http.Client
}

// tronTxInfoResponse represents the TronGrid/TronAPI response for transaction info
type tronTxInfoResponse struct {
	ID              string `json:"id"`
	BlockNumber     int64  `json:"blockNumber,omitempty"`
	BlockTimeStamp  int64  `json:"blockTimeStamp,omitempty"`
	ContractResult  []string `json:"contractResult,omitempty"`
	Receipt         *tronReceipt `json:"receipt,omitempty"`
	Result          string `json:"result,omitempty"` // "FAILED" or empty for success
}

// tronReceipt represents the receipt in a TRON transaction
type tronReceipt struct {
	Result        string `json:"result,omitempty"` // "SUCCESS" or "FAILED"
	NetFee        int64  `json:"net_fee,omitempty"`
	EnergyFee     int64  `json:"energy_fee,omitempty"`
	EnergyUsage   int64  `json:"energy_usage,omitempty"`
}

// NewTron creates a new Tron RPC client
func NewTron(baseURL string) (*Tron, error) {
	client := &http.Client{
		Timeout: tronDefaultTimeout,
	}

	return &Tron{
		baseURL: baseURL,
		client:  client,
	}, nil
}

// GetTxStatus retrieves the on-chain status of a TRON transaction by its hash.
// Returns TxOnChainPending if the transaction is not yet confirmed,
// TxOnChainSuccess if the transaction succeeded,
// TxOnChainFail if the transaction failed,
// or an error if the HTTP request or response parsing fails.
func (t *Tron) GetTxStatus(ctx context.Context, txHash string) (TxOnChainStatus, error) {
	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	// TronGrid API endpoint for transaction info
	url := fmt.Sprintf("%s/wallet/gettransactioninfobyid?value=%s", t.baseURL, txHash)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := t.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("tron API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	// Empty response or {} means transaction not found
	if len(body) == 0 || string(body) == "{}" {
		return TxOnChainPending, nil
	}

	var txResp tronTxInfoResponse
	if err := json.Unmarshal(body, &txResp); err != nil {
		return TxOnChainPending, nil
	}

	// If we don't have an ID, the transaction wasn't found
	if txResp.ID == "" {
		return TxOnChainPending, nil
	}

	// Check if the transaction is in a block
	if txResp.BlockNumber == 0 {
		return TxOnChainPending, nil
	}

	// Check the result field
	if txResp.Result == "FAILED" {
		return TxOnChainFail, nil
	}

	// Check the receipt result if available
	if txResp.Receipt != nil && txResp.Receipt.Result == "FAILED" {
		return TxOnChainFail, nil
	}

	// Transaction is confirmed and successful
	return TxOnChainSuccess, nil
}

