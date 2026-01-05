package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const cosmosDefaultTimeout = 30 * time.Second

// Cosmos implements the Rpc interface for Cosmos (Gaia) chain
type Cosmos struct {
	baseURL string
	client  *http.Client
}

// cosmosLCDTxResponse represents the Cosmos LCD API response for transaction query
type cosmosLCDTxResponse struct {
	TxResponse struct {
		Height    string `json:"height"`
		TxHash    string `json:"txhash"`
		Code      int    `json:"code"` // 0 = success, non-zero = failure
		RawLog    string `json:"raw_log,omitempty"`
		GasWanted string `json:"gas_wanted"`
		GasUsed   string `json:"gas_used"`
	} `json:"tx_response"`
}

// NewCosmos creates a new Cosmos RPC client
func NewCosmos(baseURL string) (*Cosmos, error) {
	client := &http.Client{
		Timeout: cosmosDefaultTimeout,
	}

	return &Cosmos{
		baseURL: baseURL,
		client:  client,
	}, nil
}

// GetTxStatus retrieves the on-chain status of a Cosmos transaction by its hash.
// Returns TxOnChainPending if the transaction is not yet found,
// TxOnChainSuccess if the transaction succeeded (code 0),
// TxOnChainFail if the transaction failed (code != 0),
// or an error if the HTTP request or response parsing fails.
func (c *Cosmos) GetTxStatus(ctx context.Context, txHash string) (TxOnChainStatus, error) {
	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	// Cosmos LCD API endpoint for transaction query
	url := fmt.Sprintf("%s/cosmos/tx/v1beta1/txs/%s", c.baseURL, txHash)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// 404 or similar means transaction not found - treat as pending
	if resp.StatusCode == http.StatusNotFound {
		return TxOnChainPending, nil
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("cosmos API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	var txResp cosmosLCDTxResponse
	if err := json.Unmarshal(body, &txResp); err != nil {
		// If we can't parse the response, the transaction might not exist yet
		return TxOnChainPending, nil
	}

	// Check if we got a valid response with a transaction hash
	if txResp.TxResponse.TxHash == "" {
		return TxOnChainPending, nil
	}

	// Code 0 means success in Cosmos
	if txResp.TxResponse.Code == 0 {
		return TxOnChainSuccess, nil
	}

	// Non-zero code means the transaction failed
	return TxOnChainFail, nil
}

