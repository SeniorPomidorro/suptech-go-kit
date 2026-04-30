package zoom

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/SeniorPomidorro/suptech-go-kit/pkg/transport"
)

// testEnv wires a Zoom client to an httptest server that serves both /oauth/token and arbitrary API paths.
type testEnv struct {
	server          *httptest.Server
	tokenCalls      atomic.Int64
	apiCalls        atomic.Int64
	tokenHandler    func(w http.ResponseWriter, r *http.Request)
	apiHandler      func(w http.ResponseWriter, r *http.Request)
	lastTokenAuth   string
	lastBearerAuth  string
	advertiseAPIURL string // value to put in api_url field of token response (defaults to server URL)
}

// newTestEnv builds an env. Pass nil for handlers to use defaults (token = 1h valid, API = 200 {}).
// pinAPIURL=true sets WithAPIURL so requests go to server regardless of api_url returned by token.
func newTestEnv(t *testing.T, pinAPIURL bool) (*Client, *testEnv) {
	t.Helper()
	env := &testEnv{}

	mux := http.NewServeMux()
	mux.HandleFunc("/oauth/token", func(w http.ResponseWriter, r *http.Request) {
		env.tokenCalls.Add(1)
		env.lastTokenAuth = r.Header.Get("Authorization")
		if env.tokenHandler != nil {
			env.tokenHandler(w, r)
			return
		}
		apiURL := env.advertiseAPIURL
		if apiURL == "" {
			apiURL = env.server.URL
		}
		_, _ = io.WriteString(w, fmt.Sprintf(
			`{"access_token":"tok-%d","token_type":"bearer","expires_in":3600,"scope":"meeting:write:meeting:admin","api_url":%q}`,
			env.tokenCalls.Load(), apiURL,
		))
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		env.apiCalls.Add(1)
		env.lastBearerAuth = r.Header.Get("Authorization")
		if env.apiHandler != nil {
			env.apiHandler(w, r)
			return
		}
		_, _ = io.WriteString(w, `{}`)
	})

	env.server = httptest.NewServer(mux)
	t.Cleanup(env.server.Close)

	tr := transport.New(transport.WithRetry(transport.RetryConfig{MaxAttempts: 1}))
	opts := []Option{
		WithS2SAuth("acc-id", "client-id", "client-secret"),
		WithOAuthURL(env.server.URL + "/oauth/token"),
		WithTransport(tr),
	}
	if pinAPIURL {
		opts = append(opts, WithAPIURL(env.server.URL))
	}

	c, err := NewClient(opts...)
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	return c, env
}

func TestTokenSource_FetchAndCache(t *testing.T) {
	t.Parallel()
	c, env := newTestEnv(t, true)
	ctx := context.Background()

	tok1, err := c.tokens.Token(ctx)
	if err != nil {
		t.Fatalf("first Token: %v", err)
	}
	if tok1 == "" {
		t.Fatal("token is empty")
	}
	if got := env.tokenCalls.Load(); got != 1 {
		t.Fatalf("token calls after first fetch: want 1, got %d", got)
	}

	tok2, err := c.tokens.Token(ctx)
	if err != nil {
		t.Fatalf("second Token: %v", err)
	}
	if tok2 != tok1 {
		t.Fatalf("cached token mismatch: %q vs %q", tok1, tok2)
	}
	if got := env.tokenCalls.Load(); got != 1 {
		t.Fatalf("token endpoint hit on cached read: got %d calls", got)
	}
}

func TestTokenSource_RefreshesNearExpiry(t *testing.T) {
	t.Parallel()
	c, env := newTestEnv(t, true)
	// Override: 5s expiry — well below refreshBuffer (1m), so cache treats it as expired immediately.
	env.tokenHandler = func(w http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(w, `{"access_token":"short","token_type":"bearer","expires_in":5,"api_url":"`+env.server.URL+`"}`)
	}
	ctx := context.Background()

	if _, err := c.tokens.Token(ctx); err != nil {
		t.Fatalf("first Token: %v", err)
	}
	if _, err := c.tokens.Token(ctx); err != nil {
		t.Fatalf("second Token: %v", err)
	}
	if got := env.tokenCalls.Load(); got != 2 {
		t.Fatalf("expected refresh on near-expiry: want 2 calls, got %d", got)
	}
}

func TestTokenSource_Invalidate(t *testing.T) {
	t.Parallel()
	c, env := newTestEnv(t, true)
	ctx := context.Background()

	if _, err := c.tokens.Token(ctx); err != nil {
		t.Fatalf("Token: %v", err)
	}
	c.tokens.Invalidate()
	if _, err := c.tokens.Token(ctx); err != nil {
		t.Fatalf("Token after invalidate: %v", err)
	}
	if got := env.tokenCalls.Load(); got != 2 {
		t.Fatalf("Invalidate did not force refetch: got %d calls", got)
	}
}

func TestTokenSource_BasicAuthHeader(t *testing.T) {
	t.Parallel()
	c, env := newTestEnv(t, true)
	if _, err := c.tokens.Token(context.Background()); err != nil {
		t.Fatalf("Token: %v", err)
	}
	want := "Basic " + base64.StdEncoding.EncodeToString([]byte("client-id:client-secret"))
	if env.lastTokenAuth != want {
		t.Fatalf("token request auth header mismatch: want %q, got %q", want, env.lastTokenAuth)
	}
}

func TestClient_APIURLFromTokenResponse(t *testing.T) {
	t.Parallel()
	// Don't pin via WithAPIURL — let client learn it from api_url field.
	c, env := newTestEnv(t, false)
	ctx := context.Background()

	// First call provokes token fetch + api request.
	if _, err := c.Users().Get(ctx, "test@example.com"); err != nil {
		t.Fatalf("Users().Get: %v", err)
	}
	if got := env.apiCalls.Load(); got != 1 {
		t.Fatalf("api calls: want 1, got %d", got)
	}
}

func TestClient_WithAPIURLOverridesTokenAPIURL(t *testing.T) {
	t.Parallel()
	// api_url advertised by token endpoint is bogus; WithAPIURL should win and we still hit the test server.
	c, env := newTestEnv(t, true)
	env.advertiseAPIURL = "https://nowhere.invalid"
	ctx := context.Background()

	if _, err := c.Users().Get(ctx, "test@example.com"); err != nil {
		t.Fatalf("Users().Get: %v", err)
	}
	if got := env.apiCalls.Load(); got != 1 {
		t.Fatalf("api calls: want 1, got %d", got)
	}
}

func TestClient_BearerHeaderInjected(t *testing.T) {
	t.Parallel()
	c, env := newTestEnv(t, true)
	if _, err := c.Users().Get(context.Background(), "test@example.com"); err != nil {
		t.Fatalf("Users().Get: %v", err)
	}
	if !strings.HasPrefix(env.lastBearerAuth, "Bearer tok-") {
		t.Fatalf("api request bearer header mismatch: got %q", env.lastBearerAuth)
	}
}

func TestClient_RefreshesAndRetriesOn401InvalidToken(t *testing.T) {
	t.Parallel()
	c, env := newTestEnv(t, true)
	// Fail first API request with 401+code 124, succeed on second.
	env.apiHandler = func(w http.ResponseWriter, _ *http.Request) {
		if env.apiCalls.Load() == 1 {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = io.WriteString(w, `{"code":124,"message":"Invalid access token."}`)
			return
		}
		_, _ = io.WriteString(w, `{"id":"u1","email":"x@y.z"}`)
	}

	user, err := c.Users().Get(context.Background(), "x@y.z")
	if err != nil {
		t.Fatalf("Users().Get: %v", err)
	}
	if user.Email != "x@y.z" {
		t.Fatalf("decoded user email: want x@y.z, got %q", user.Email)
	}
	if got := env.apiCalls.Load(); got != 2 {
		t.Fatalf("api calls: want 2 (initial + retry), got %d", got)
	}
	if got := env.tokenCalls.Load(); got != 2 {
		t.Fatalf("token calls: want 2 (initial + force-refresh), got %d", got)
	}
}

func TestClient_DoesNotRetryOn401MissingScope(t *testing.T) {
	t.Parallel()
	c, env := newTestEnv(t, true)
	env.apiHandler = func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, `{"code":4711,"message":"Invalid access token, does not contain scopes:[meeting:delete:meeting:admin]"}`)
	}

	_, err := c.Users().Get(context.Background(), "x@y.z")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsMissingScope(err) {
		t.Fatalf("expected IsMissingScope, got: %v", err)
	}
	if IsInvalidToken(err) {
		t.Fatalf("missing-scope error should not match IsInvalidToken: %v", err)
	}
	if got := env.apiCalls.Load(); got != 1 {
		t.Fatalf("api calls: want 1 (no retry), got %d", got)
	}
}

func TestClient_DoesNotRetryAfterSecond401(t *testing.T) {
	t.Parallel()
	c, env := newTestEnv(t, true)
	// Always fail with code 124 — second attempt must surface error, not loop forever.
	env.apiHandler = func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = io.WriteString(w, `{"code":124,"message":"Invalid access token."}`)
	}

	_, err := c.Users().Get(context.Background(), "x@y.z")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !IsInvalidToken(err) {
		t.Fatalf("expected IsInvalidToken, got: %v", err)
	}
	if got := env.apiCalls.Load(); got != 2 {
		t.Fatalf("api calls: want exactly 2 (initial + one retry), got %d", got)
	}
}

func TestErrors_HelpersOnNonZoomError(t *testing.T) {
	t.Parallel()
	if IsInvalidToken(nil) || IsMissingScope(nil) {
		t.Fatal("nil error must not match any classifier")
	}
	plain := errors.New("plain error")
	if IsInvalidToken(plain) || IsMissingScope(plain) {
		t.Fatal("non-zoom error must not match any classifier")
	}
}

func TestErrors_HelpersOnZoomCodes(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name           string
		code           int
		message        string
		wantInvalid    bool
		wantMissingSc  bool
	}{
		{"invalid token", ErrCodeInvalidAccessToken, "Invalid access token.", true, false},
		{"missing scope by code", ErrCodeMissingScope, "anything", false, true},
		{"missing scope by message", 1234, "does not contain scopes:[foo]", false, true},
		{"unrelated", 5555, "something else", false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := &Error{Code: tc.code, Message: tc.message}
			if got := IsInvalidToken(err); got != tc.wantInvalid {
				t.Errorf("IsInvalidToken: want %v, got %v", tc.wantInvalid, got)
			}
			if got := IsMissingScope(err); got != tc.wantMissingSc {
				t.Errorf("IsMissingScope: want %v, got %v", tc.wantMissingSc, got)
			}
		})
	}
}

// Sanity check that a refresh that fails surfaces a proper error (not a panic / not a stale token).
func TestTokenSource_RefreshFailureSurfaces(t *testing.T) {
	t.Parallel()
	c, env := newTestEnv(t, true)
	env.tokenHandler = func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = io.WriteString(w, `{"reason":"Invalid client_id or client_secret"}`)
	}
	_, err := c.tokens.Token(context.Background())
	if err == nil {
		t.Fatal("expected error from failed refresh")
	}
	var ze *Error
	if !errors.As(err, &ze) {
		t.Fatalf("expected *zoom.Error, got %T: %v", err, err)
	}
	if ze.StatusCode != http.StatusBadRequest {
		t.Fatalf("status code: want 400, got %d", ze.StatusCode)
	}
}

