package portal

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type ListingFeeResponse struct {
	PolicyID   string  `json:"policy_id"`
	PublicKey  string  `json:"public_key"`
	PluginID   string  `json:"target_plugin_id"`
	Status     string  `json:"status"`
	TxHash     *string `json:"tx_hash"`
	PaidAt     *string `json:"paid_at"`
	FailedAt   *string `json:"failed_at"`
	FailReason *string `json:"failure_reason"`
}

type ListingFeeClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewListingFeeClient(baseURL string) *ListingFeeClient {
	return &ListingFeeClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *ListingFeeClient) GetListingFeeByScope(ctx context.Context, pubkey, pluginID string) (*ListingFeeResponse, error) {
	u, err := url.Parse(c.baseURL + "/api/listing-fee/by-scope")
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	q := u.Query()
	q.Set("pubkey", pubkey)
	q.Set("pluginId", pluginID)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result ListingFeeResponse
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	return &result, nil
}

func (c *ListingFeeClient) IsListingFeePaid(ctx context.Context, pluginID string) (bool, error) {
	u, err := url.Parse(c.baseURL + "/api/listing-fee/paid")
	if err != nil {
		return false, fmt.Errorf("invalid base URL: %w", err)
	}

	q := u.Query()
	q.Set("pluginId", pluginID)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return false, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result struct {
		Paid bool `json:"paid"`
	}
	err = json.NewDecoder(resp.Body).Decode(&result)
	if err != nil {
		return false, fmt.Errorf("decode response: %w", err)
	}

	return result.Paid, nil
}
