package keysign

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/vultisig/verifier/plugin/libhttp"
	"github.com/vultisig/verifier/types"
)

func NewVerifierEmitter(url, token string) Emitter {
	return newApiEmitter[string](
		http.MethodPost,
		url+"/plugin-signer/sign",
		map[string]string{
			"Authorization": "Bearer " + token,
			"Content-Type":  "application/json",
		},
	)
}

type apiEmitter[T comparable] struct {
	method   string
	endpoint string
	headers  map[string]string
}

// T is response type from the HTTP API call
func newApiEmitter[T comparable](method, endpoint string, headers map[string]string) *apiEmitter[T] {
	return &apiEmitter[T]{
		method:   method,
		endpoint: endpoint,
		headers:  headers,
	}
}

func (e *apiEmitter[T]) Sign(ctx context.Context, req types.PluginKeysignRequest) error {
	_, err := libhttp.Call[T](ctx, e.method, e.endpoint, e.headers, req, nil)
	if err != nil {
		var httpErr *libhttp.HTTPError
		if errors.As(err, &httpErr) && httpErr.StatusCode == http.StatusLocked {
			return ErrPluginPaused
		}
		return fmt.Errorf("failed to make API call: %w", err)
	}
	return nil
}
