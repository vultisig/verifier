package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const utxoDefaultTimeout = 30 * time.Second

// Utxo is a generic UTXO chain RPC client that uses Blockchair API.
// It supports any UTXO chain that Blockchair provides (litecoin, dogecoin, bitcoin-cash, etc.)
type Utxo struct {
	baseURL   string
	chainPath string // e.g., "litecoin", "dogecoin", "bitcoin-cash"
	client    *http.Client
}

// NewUtxo creates a new UTXO RPC client for the specified chain.
// chainPath should be the Blockchair API path for the chain (e.g., "litecoin", "dogecoin", "bitcoin-cash")
func NewUtxo(baseURL, chainPath string) (*Utxo, error) {
	client := &http.Client{
		Timeout: utxoDefaultTimeout,
	}

	return &Utxo{
		baseURL:   baseURL,
		chainPath: chainPath,
		client:    client,
	}, nil
}

// GetTxStatus retrieves the on-chain status of a transaction by its hash.
// Returns TxOnChainPending if the transaction is not yet confirmed or not found,
// TxOnChainSuccess if the transaction is confirmed with at least one confirmation,
// or an error if the HTTP request or response parsing fails.
func (u *Utxo) GetTxStatus(ctx context.Context, txHash string) (TxOnChainStatus, error) {
	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	url := fmt.Sprintf("%s/%s/dashboards/transaction/%s", u.baseURL, u.chainPath, txHash)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := u.client.Do(req)
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

// NewLitecoin creates a Litecoin RPC client using Blockchair
func NewLitecoin(baseURL string) (*Utxo, error) {
	return NewUtxo(baseURL, "litecoin")
}

// NewDogecoin creates a Dogecoin RPC client using Blockchair
func NewDogecoin(baseURL string) (*Utxo, error) {
	return NewUtxo(baseURL, "dogecoin")
}

// NewBitcoinCash creates a Bitcoin Cash RPC client using Blockchair
func NewBitcoinCash(baseURL string) (*Utxo, error) {
	return NewUtxo(baseURL, "bitcoin-cash")
}
