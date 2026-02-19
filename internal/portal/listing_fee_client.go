package portal

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

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
