package libhttp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type respStruct struct {
	Message string `json:"message"`
	N       int    `json:"n"`
}

func TestCall_String(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("hello world"))
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	got, err := Call[string](ctx, http.MethodGet, srv.URL, nil, nil, nil)
	if err != nil {
		t.Fatalf("Call[string] error: %v", err)
	}
	if got != "hello world" {
		t.Fatalf("want %q, got %q", "hello world", got)
	}
}

func TestCall_JSONStruct(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(respStruct{Message: "ok", N: 42})
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	got, err := Call[respStruct](ctx, http.MethodGet, srv.URL, nil, nil, nil)
	if err != nil {
		t.Fatalf("Call[respStruct] error: %v", err)
	}
	if got.Message != "ok" || got.N != 42 {
		t.Fatalf("unexpected: %#v", got)
	}
}
