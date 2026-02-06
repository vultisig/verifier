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

const xrpDefaultTimeout = 30 * time.Second

type XRP struct {
	rpcURL string
	client *http.Client
}

type XRPTransactionResponse struct {
	Result struct {
		TransactionIndex int                    `json:"TransactionIndex"`
		Meta             XRPTransactionMeta     `json:"meta"`
		Transaction      map[string]interface{} `json:"transaction"`
		Validated        bool                   `json:"validated"`
	} `json:"result"`
	Status string      `json:"status"`
	Type   string      `json:"type"`
	Error  interface{} `json:"error,omitempty"`
}

type XRPTransactionMeta struct {
	TransactionResult string `json:"TransactionResult"`
}

type XRPRequest struct {
	Method string `json:"method"`
	Params []any  `json:"params"`
}

type XRPTransactionParams struct {
	Transaction string `json:"transaction"`
	Binary      bool   `json:"binary"`
}

func NewXRP(rpcURL string) (*XRP, error) {
	client := &http.Client{
		Timeout: xrpDefaultTimeout,
	}

	// Test connection by making a simple request
	xrp := &XRP{
		rpcURL: rpcURL,
		client: client,
	}

	// Test with a simple server_info request
	testReq := XRPRequest{
		Method: "server_info",
		Params: []any{},
	}

	_, err := xrp.makeRequest(context.Background(), testReq)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to XRP RPC: %w", err)
	}

	return xrp, nil
}

func (x *XRP) GetTxStatus(ctx context.Context, txHash string) (TxStatusResult, error) {
	if ctx.Err() != nil {
		return TxStatusResult{}, ctx.Err()
	}

	req := XRPRequest{
		Method: "tx",
		Params: []any{
			XRPTransactionParams{
				Transaction: txHash,
				Binary:      false,
			},
		},
	}

	respBody, err := x.makeRequest(ctx, req)
	if err != nil {
		return TxStatusResult{}, fmt.Errorf("failed to get transaction status: %w", err)
	}

	var txResp XRPTransactionResponse
	if err := json.Unmarshal(respBody, &txResp); err != nil {
		return TxStatusResult{}, fmt.Errorf("failed to unmarshal XRP transaction response: %w", err)
	}

	if txResp.Error != nil {
		return NewTxStatusResult(TxOnChainPending, ""), nil
	}

	if !txResp.Result.Validated {
		return NewTxStatusResult(TxOnChainPending, ""), nil
	}

	switch txResp.Result.Meta.TransactionResult {
	case "tesSUCCESS":
		return NewTxStatusResult(TxOnChainSuccess, ""), nil
	default:
		return NewTxStatusResult(TxOnChainFail, txResp.Result.Meta.TransactionResult), nil
	}
}

func (x *XRP) makeRequest(ctx context.Context, req XRPRequest) ([]byte, error) {
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", x.rpcURL, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := x.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to make HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("XRP RPC returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return body, nil
}
