package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const zcashDefaultTimeout = 30 * time.Second

type Zcash struct {
	baseURL string
	client  *http.Client
}

// blockchairTxResponse represents the Blockchair API response for transaction info
type blockchairTxResponse struct {
	Data map[string]struct {
		Transaction struct {
			BlockID       int    `json:"block_id"`
			Hash          string `json:"hash"`
			IsConfirmed   bool   `json:"is_confirmed"`
			Confirmations int    `json:"confirmations"`
		} `json:"transaction"`
	} `json:"data"`
	Context struct {
		Code  int    `json:"code"`
		Error string `json:"error,omitempty"`
	} `json:"context"`
}

func NewZcash(baseURL string) (*Zcash, error) {
	client := &http.Client{
		Timeout: zcashDefaultTimeout,
	}

	return &Zcash{
		baseURL: baseURL,
		client:  client,
	}, nil
}

func (z *Zcash) GetTxStatus(ctx context.Context, txHash string) (TxOnChainStatus, error) {
	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	url := fmt.Sprintf("%s/zcash/dashboards/transaction/%s", z.baseURL, txHash)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := z.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// 404 or similar means transaction not found - treat as pending
	if resp.StatusCode == http.StatusNotFound {
		return TxOnChainPending, nil
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("blockchair API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	var txResp blockchairTxResponse
	if err := json.Unmarshal(body, &txResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Check for API-level errors
	if txResp.Context.Code != 200 || txResp.Context.Error != "" {
		// Transaction not found or other error - treat as pending
		return TxOnChainPending, nil
	}

	// Check if transaction data exists
	txData, exists := txResp.Data[txHash]
	if !exists {
		return TxOnChainPending, nil
	}

	// Check confirmations
	if txData.Transaction.BlockID == 0 || txData.Transaction.Confirmations == 0 {
		return TxOnChainPending, nil
	}

	// Transaction is confirmed
	return TxOnChainSuccess, nil
}

