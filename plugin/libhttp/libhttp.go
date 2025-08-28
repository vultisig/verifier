package libhttp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	stdurl "net/url"
	"time"
)

func Call[T comparable](
	ctx context.Context,
	method, url string,
	headers map[string]string,
	body interface{},
	query map[string]string,
) (T, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return *new(T), fmt.Errorf("failed to marshal request json: %w", err)
	}

	var q string
	if query != nil {
		qurl := stdurl.Values{}
		for k, v := range query {
			qurl.Set(k, v)
		}
		q = "?" + qurl.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, method, url+q, bytes.NewReader(b))
	if err != nil {
		return *new(T), fmt.Errorf("failed to build http request: %w", err)
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	res, err := client.Do(req)
	if err != nil {
		return *new(T), fmt.Errorf("failed to make http call: %w", err)
	}
	defer func() {
		_ = res.Body.Close()
	}()

	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return *new(T), fmt.Errorf("failed to read response body: %w", err)
	}
	if res.StatusCode != http.StatusOK {
		return *new(T), fmt.Errorf("failed to get successful response: status_code: %d, res_body: %s", res.StatusCode, string(bodyBytes))
	}

	if _, ok := any(*new(T)).(string); ok {
		return any(string(bodyBytes)).(T), nil
	}

	var r T
	err = json.Unmarshal(bodyBytes, &r)
	if err != nil {
		return *new(T), fmt.Errorf("failed to unmarshal response json: %w", err)
	}

	return r, nil
}
