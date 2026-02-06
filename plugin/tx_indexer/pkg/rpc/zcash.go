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

// GetTxStatus retrieves the on-chain status of a Zcash transaction by its hash.
// Returns TxOnChainPending if the transaction is not yet confirmed or not found,
// TxOnChainSuccess if the transaction is confirmed with at least one confirmation,
// or an error if the HTTP request or response parsing fails.
// Note: This client does not attempt to extract/store failure reasons for UTXO chains;
// it only returns pending vs confirmed. UTXO broadcast/mempool rejection reasons aren't captured here.
func (z *Zcash) GetTxStatus(ctx context.Context, txHash string) (TxStatusResult, error) {
	if ctx.Err() != nil {
		return TxStatusResult{}, ctx.Err()
	}

	url := fmt.Sprintf("%s/zcash/dashboards/transaction/%s", z.baseURL, txHash)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return TxStatusResult{}, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := z.client.Do(req)
	if err != nil {
		return TxStatusResult{}, fmt.Errorf("failed to make request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return NewTxStatusResult(TxOnChainPending, ""), nil
	}

	if resp.StatusCode != http.StatusOK {
		return TxStatusResult{}, fmt.Errorf("blockchair API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return TxStatusResult{}, fmt.Errorf("failed to read response body: %w", err)
	}

	var txResp blockchairTxResponse
	if err := json.Unmarshal(body, &txResp); err != nil {
		return TxStatusResult{}, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if txResp.Context.Code != 200 || txResp.Context.Error != "" {
		if txResp.Context.Code == 404 {
			return NewTxStatusResult(TxOnChainPending, ""), nil
		}
		return TxStatusResult{}, fmt.Errorf("blockchair API error: code=%d, error=%s", txResp.Context.Code, txResp.Context.Error)
	}

	txData, exists := txResp.Data[txHash]
	if !exists {
		return NewTxStatusResult(TxOnChainPending, ""), nil
	}

	if txData.Transaction.BlockID == 0 || txData.Transaction.Confirmations == 0 {
		return NewTxStatusResult(TxOnChainPending, ""), nil
	}

	return NewTxStatusResult(TxOnChainSuccess, ""), nil
}
