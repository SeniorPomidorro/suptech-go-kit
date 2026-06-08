package transport

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
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

// slowFirstAttemptServer returns a server whose first request blocks past the
// client timeout and whose subsequent requests respond immediately, plus a
// func reporting how many requests it received.
func slowFirstAttemptServer(t *testing.T, firstDelay time.Duration) (*httptest.Server, func() int) {
	t.Helper()
	var mu sync.Mutex
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		hits++
		n := hits
		mu.Unlock()
		if n == 1 {
			time.Sleep(firstDelay)
		}
		w.WriteHeader(http.StatusOK)
	}))
	return srv, func() int {
		mu.Lock()
		defer mu.Unlock()
		return hits
	}
}

func TestDoDoesNotRetryClientTimeoutByDefault(t *testing.T) {
	t.Parallel()

	srv, hits := slowFirstAttemptServer(t, 500*time.Millisecond)
	defer srv.Close()

	client := New(
		WithTimeout(100*time.Millisecond),
		WithRetry(RetryConfig{
			MaxAttempts:    3,
			InitialBackoff: time.Millisecond,
			MaxBackoff:     time.Millisecond,
			// RetryClientTimeout defaults to false.
		}),
	)

	req, err := http.NewRequest(http.MethodGet, srv.URL, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	resp, err := client.Do(req)
	if err == nil {
		_ = resp.Body.Close()
		t.Fatalf("expected client timeout error, got nil")
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected DeadlineExceeded, got %v", err)
	}
	if got := hits(); got != 1 {
		t.Fatalf("expected exactly 1 attempt (no retry by default), got %d", got)
	}
}

func TestDoRetriesClientTimeoutWhenEnabled(t *testing.T) {
	t.Parallel()

	srv, hits := slowFirstAttemptServer(t, 500*time.Millisecond)
	defer srv.Close()

	client := New(
		WithTimeout(100*time.Millisecond),
		WithRetry(RetryConfig{
			MaxAttempts:        3,
			InitialBackoff:     time.Millisecond,
			MaxBackoff:         time.Millisecond,
			RetryClientTimeout: true,
		}),
	)

	req, err := http.NewRequest(http.MethodGet, srv.URL, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("expected success after retrying timeout, got %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if got := hits(); got != 2 {
		t.Fatalf("expected 2 attempts (timeout then success), got %d", got)
	}
}

func TestDoDoesNotRetryCallerDeadlineEvenWhenTimeoutRetryEnabled(t *testing.T) {
	t.Parallel()

	srv, hits := slowFirstAttemptServer(t, 500*time.Millisecond)
	defer srv.Close()

	// No client-level timeout: the only deadline is the caller's context.
	client := New(WithRetry(RetryConfig{
		MaxAttempts:        3,
		InitialBackoff:     time.Millisecond,
		MaxBackoff:         time.Millisecond,
		RetryClientTimeout: true, // opt-in must NOT override caller cancellation
	}))

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}

	resp, err := client.Do(req)
	if err == nil {
		_ = resp.Body.Close()
		t.Fatalf("expected caller-deadline error, got nil")
	}
	if got := hits(); got != 1 {
		t.Fatalf("expected exactly 1 attempt (caller deadline, no retry), got %d", got)
	}
}

// TestShouldRetryErrorClassification pins the full truth table of the retry
// decision at the unit level. It is the only coverage of the branch that
// actually exercises the ctx.Err() guard: a cancelled caller context paired
// with a generic, retryable net.Error must NOT be retried. The Do-level tests
// above cannot defend that guard — their errors are context.DeadlineExceeded,
// which sleepWithContext would also bail on, so they stay green even if the
// guard is deleted.
func TestShouldRetryErrorClassification(t *testing.T) {
	t.Parallel()

	netErr := &net.OpError{Op: "read", Err: errors.New("connection reset by peer")}
	done, cancel := context.WithCancel(context.Background())
	cancel()
	alive := context.Background()

	cases := []struct {
		name         string
		retryTimeout bool
		ctx          context.Context
		err          error
		want         bool
	}{
		{"nil error", true, alive, nil, false},
		{"caller cancelled + net.Error: honor cancellation", true, done, netErr, false},
		{"caller cancelled + Canceled err", true, done, context.Canceled, false},
		{"live ctx + transient net.Error: retry", true, alive, netErr, true},
		{"live ctx + client timeout, flag on: retry", true, alive, context.DeadlineExceeded, true},
		{"live ctx + client timeout, flag off: no retry", false, alive, context.DeadlineExceeded, false},
		{"caller deadline done + DeadlineExceeded, flag on: honor cancellation", true, done, context.DeadlineExceeded, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := New(WithRetry(RetryConfig{RetryClientTimeout: tc.retryTimeout}))
			if got := c.shouldRetryError(tc.ctx, tc.err); got != tc.want {
				t.Fatalf("shouldRetryError(ctx, %v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}
