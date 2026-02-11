package transport

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestDoJSONRetriesOnTransientStatus(t *testing.T) {
	t.Parallel()

	attempt := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempt++
		if attempt < 3 {
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte("temporary"))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	client := New(WithRetry(RetryConfig{
		MaxAttempts:    3,
		InitialBackoff: time.Millisecond,
		MaxBackoff:     time.Millisecond,
	}))

	req, err := http.NewRequest(http.MethodGet, srv.URL, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	var out struct {
		OK bool `json:"ok"`
	}
	if err := client.DoJSON(req, &out); err != nil {
		t.Fatalf("DoJSON failed: %v", err)
	}
	if !out.OK {
		t.Fatalf("expected ok=true")
	}
	if attempt != 3 {
		t.Fatalf("expected 3 attempts, got %d", attempt)
	}
}

func TestDoJSONReturnsAPIErrorWithLimitedBody(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(strings.Repeat("a", 128)))
	}))
	defer srv.Close()

	client := New(WithErrorBodyLimit(16))
	req, err := http.NewRequest(http.MethodGet, srv.URL, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	err = client.DoJSON(req, &struct{}{})
	if err == nil {
		t.Fatalf("expected error")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got %T", err)
	}
	if apiErr.StatusCode != http.StatusBadRequest {
		t.Fatalf("unexpected status code: %d", apiErr.StatusCode)
	}
	if len(apiErr.Body) != 16 {
		t.Fatalf("expected limited body length 16, got %d", len(apiErr.Body))
	}
}

func TestDoAppliesBaseHeaders(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-Test"); got != "abc" {
			t.Fatalf("expected X-Test header abc, got %q", got)
		}
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	headers := http.Header{"X-Test": []string{"abc"}}
	client := New(WithBaseHeaders(headers))
	req, err := http.NewRequest(http.MethodGet, srv.URL, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Do failed: %v", err)
	}
	_ = resp.Body.Close()
}
