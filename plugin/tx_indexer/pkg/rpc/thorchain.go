package rpc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
)

type THORChain struct {
	client  *http.Client
	baseURL string
	limiter *rate.Limiter
}

func NewTHORChain(rpcURL string) (*THORChain, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Clean up URL
	baseURL := strings.TrimSuffix(rpcURL, "/")

	// Rate limiter: 1 req/s with burst of 3 (more lenient)
	limiter := rate.NewLimiter(rate.Limit(1), 3)

	return &THORChain{
		client:  client,
		baseURL: baseURL,
		limiter: limiter,
	}, nil
}

// THORChain Tendermint RPC transaction response structure
type thorchainTxResponse struct {
	Result struct {
		Hash     string `json:"hash"`
		Height   string `json:"height"`
		TxResult struct {
			Code int `json:"code"`
		} `json:"tx_result"`
	} `json:"result"`
	Error *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func (t *THORChain) GetTxStatus(ctx context.Context, txHash string) (TxOnChainStatus, error) {
	// Use Tendermint RPC format: prefix with 0x and ensure uppercase
	formattedHash := strings.ToUpper(strings.TrimPrefix(txHash, "0x"))
	url := fmt.Sprintf("%s/tx?hash=0x%s", t.baseURL, formattedHash)

	return t.makeRequestWithRetry(ctx, url)
}

func (t *THORChain) makeRequestWithRetry(ctx context.Context, url string) (TxOnChainStatus, error) {
	maxRetries := 3
	baseDelay := 2 * time.Second

	for attempt := 0; attempt <= maxRetries; attempt++ {
		// Rate limiting
		if err := t.limiter.Wait(ctx); err != nil {
			return TxOnChainFail, fmt.Errorf("rate limiter context error: %w", err)
		}

		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return TxOnChainFail, fmt.Errorf("failed to create request: %w", err)
		}

		resp, err := t.client.Do(req)
		if err != nil {
			if attempt == maxRetries {
				return TxOnChainFail, fmt.Errorf("failed to make request after %d attempts: %w", maxRetries+1, err)
			}
			delay := t.calculateBackoff(attempt, baseDelay)
			logrus.WithFields(logrus.Fields{
				"attempt":       attempt + 1,
				"max_attempts":  maxRetries + 1,
				"delay_seconds": delay.Seconds(),
				"error":         err.Error(),
			}).Info("THORChain RPC request failed, retrying")
			t.sleep(ctx, delay)
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return TxOnChainFail, fmt.Errorf("failed to read response: %w", err)
		}

		// Handle rate limiting (429) - don't fail, just return pending
		if resp.StatusCode == http.StatusTooManyRequests {
			if attempt == maxRetries {
				logrus.WithFields(logrus.Fields{
					"attempt":      attempt + 1,
					"max_attempts": maxRetries + 1,
				}).Warn("THORChain RPC rate limited after max retries, returning pending")
				return TxOnChainPending, nil
			}

			delay := t.getRetryAfterDelay(resp, t.calculateBackoff(attempt, baseDelay))
			t.sleep(ctx, delay)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			return TxOnChainPending, fmt.Errorf("HTTP error: %d, body: %s", resp.StatusCode, string(body))
		}

		var txResp thorchainTxResponse
		if err := json.Unmarshal(body, &txResp); err != nil {
			return TxOnChainPending, fmt.Errorf("failed to unmarshal response: %w", err)
		}

		// Check for RPC error response
		if txResp.Error != nil {
			// Transaction not found - still pending
			return TxOnChainPending, nil
		}

		// Check if transaction exists and has a height (confirmed)
		if txResp.Result.Hash != "" && txResp.Result.Height != "" {
			// Tendermint RPC: code 0 = success, non-zero = failure
			if txResp.Result.TxResult.Code == 0 {
				return TxOnChainSuccess, nil
			}
			return TxOnChainFail, nil
		}

		return TxOnChainPending, nil
	}

	// Don't crash on max retries - just return pending
	return TxOnChainPending, nil
}

func (t *THORChain) calculateBackoff(attempt int, baseDelay time.Duration) time.Duration {
	// Exponential backoff: 1s → 2s → 4s → 8s, max 30s
	delay := baseDelay * time.Duration(math.Pow(2, float64(attempt)))
	maxDelay := 30 * time.Second
	if delay > maxDelay {
		delay = maxDelay
	}

	// Add jitter (±25%)
	jitter := time.Duration(rand.Float64()*0.5-0.25) * delay
	return delay + jitter
}

func (t *THORChain) getRetryAfterDelay(resp *http.Response, fallback time.Duration) time.Duration {
	retryAfter := resp.Header.Get("Retry-After")
	if retryAfter == "" {
		return fallback
	}

	// Try parsing as seconds
	if seconds, err := strconv.Atoi(retryAfter); err == nil {
		delay := time.Duration(seconds) * time.Second
		maxDelay := 30 * time.Second
		if delay > maxDelay {
			delay = maxDelay
		}
		return delay
	}

	return fallback
}

func (t *THORChain) sleep(ctx context.Context, duration time.Duration) {
	timer := time.NewTimer(duration)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return
	case <-timer.C:
		return
	}
}
