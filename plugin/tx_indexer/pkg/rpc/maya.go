package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const mayaDefaultTimeout = 30 * time.Second

// Maya implements the Rpc interface for MayaChain
type Maya struct {
	baseURL string
	client  *http.Client
}

// mayaTxResponse represents the MayaChain API response for transaction query
// MayaChain uses the same format as THORChain
type mayaTxResponse struct {
	ObservedTx struct {
		Tx struct {
			ID string `json:"id"`
		} `json:"tx"`
		Status string `json:"status"` // "done" or "pending"
	} `json:"observed_tx"`
	FinalizedHeight int64 `json:"finalized_height,omitempty"`
}

// mayaCosmosLCDTxResponse is the Cosmos LCD format response
type mayaCosmosLCDTxResponse struct {
	TxResponse struct {
		Height string `json:"height"`
		TxHash string `json:"txhash"`
		Code   int    `json:"code"`
	} `json:"tx_response"`
}

// NewMaya creates a new Maya RPC client
func NewMaya(baseURL string) (*Maya, error) {
	client := &http.Client{
		Timeout: mayaDefaultTimeout,
	}

	return &Maya{
		baseURL: baseURL,
		client:  client,
	}, nil
}

// GetTxStatus retrieves the on-chain status of a MayaChain transaction by its hash.
// Returns TxOnChainPending if the transaction is not yet found or still pending,
// TxOnChainSuccess if the transaction is confirmed,
// TxOnChainFail if the transaction failed,
// or an error if the HTTP request or response parsing fails.
func (m *Maya) GetTxStatus(ctx context.Context, txHash string) (TxOnChainStatus, error) {
	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	// Try the Cosmos LCD API first (MayaChain is Cosmos-based)
	url := fmt.Sprintf("%s/cosmos/tx/v1beta1/txs/%s", m.baseURL, txHash)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := m.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// 404 or similar means transaction not found - treat as pending
	if resp.StatusCode == http.StatusNotFound {
		return TxOnChainPending, nil
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("maya API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	var txResp mayaCosmosLCDTxResponse
	if err := json.Unmarshal(body, &txResp); err != nil {
		return TxOnChainPending, nil
	}

	// Check if we got a valid response
	if txResp.TxResponse.TxHash == "" {
		return TxOnChainPending, nil
	}

	// Code 0 means success
	if txResp.TxResponse.Code == 0 {
		return TxOnChainSuccess, nil
	}

	return TxOnChainFail, nil
}

