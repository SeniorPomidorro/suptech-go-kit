package zoom

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/SeniorPomidorro/suptech-go-kit/pkg/transport"
)

// Defaults for Zoom S2S OAuth.
// Reference: developers.zoom.us/docs/internal-apps/s2s-oauth.
const (
	DefaultOAuthURL = "https://zoom.us/oauth/token"
	DefaultAPIURL   = "https://api.zoom.us/v2"

	// refreshBuffer makes the cached token "expired" this much before its real expiry,
	// so callers never use a token that is about to die mid-request.
	refreshBuffer = 1 * time.Minute
)

// tokenResponse mirrors the JSON returned by POST /oauth/token.
type tokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	Scope       string `json:"scope"`
	APIURL      string `json:"api_url"`
}

// tokenSource fetches and caches Zoom S2S OAuth access tokens.
//
// Behavior:
//   - Lazy refresh: token is re-requested only when a caller actually needs it
//     and the cached one is within refreshBuffer of expiry.
//   - Singleflight via mutex: when many callers race on an expired token, only
//     one performs the HTTP refresh; the rest reuse the freshly stored token.
//   - The token endpoint response carries an api_url field that is the cluster
//     base URL for subsequent API calls. We expose it via APIURL().
type tokenSource struct {
	transport    *transport.Client
	oauthURL     string
	accountID    string
	clientID     string
	clientSecret string

	mu        sync.RWMutex
	token     string
	expiresAt time.Time
	apiURL    string
}

func newTokenSource(tr *transport.Client, oauthURL, accountID, clientID, clientSecret string) (*tokenSource, error) {
	if strings.TrimSpace(accountID) == "" {
		return nil, errors.New("zoom: account ID is required for S2S auth")
	}
	if strings.TrimSpace(clientID) == "" || strings.TrimSpace(clientSecret) == "" {
		return nil, errors.New("zoom: client ID and client secret are required for S2S auth")
	}
	if tr == nil {
		tr = transport.New()
	}
	if strings.TrimSpace(oauthURL) == "" {
		oauthURL = DefaultOAuthURL
	}

	return &tokenSource{
		transport:    tr,
		oauthURL:     oauthURL,
		accountID:    accountID,
		clientID:     clientID,
		clientSecret: clientSecret,
	}, nil
}

// Token returns a valid access token, refreshing it if necessary.
func (ts *tokenSource) Token(ctx context.Context) (string, error) {
	ts.mu.RLock()
	if ts.token != "" && time.Now().Before(ts.expiresAt.Add(-refreshBuffer)) {
		t := ts.token
		ts.mu.RUnlock()
		return t, nil
	}
	ts.mu.RUnlock()

	ts.mu.Lock()
	defer ts.mu.Unlock()
	// Re-check after acquiring the write lock — another goroutine may have refreshed.
	if ts.token != "" && time.Now().Before(ts.expiresAt.Add(-refreshBuffer)) {
		return ts.token, nil
	}
	return ts.fetchLocked(ctx)
}

// Invalidate forces the next Token() call to refresh.
// Used after receiving 401 with code 124 to recover from server-side revocation.
func (ts *tokenSource) Invalidate() {
	ts.mu.Lock()
	ts.token = ""
	ts.expiresAt = time.Time{}
	ts.mu.Unlock()
}

// APIURL returns the cluster base URL learned from the latest token response.
// Empty string until the first successful refresh.
func (ts *tokenSource) APIURL() string {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	return ts.apiURL
}

func (ts *tokenSource) fetchLocked(ctx context.Context) (string, error) {
	form := url.Values{}
	form.Set("grant_type", "account_credentials")
	form.Set("account_id", ts.accountID)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, ts.oauthURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("zoom: build token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Basic "+basicAuth(ts.clientID, ts.clientSecret))

	resp, err := ts.transport.Do(req)
	if err != nil {
		return "", fmt.Errorf("zoom: fetch token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		apiErr := transport.NewAPIError(resp, 0)
		return "", asZoomError(apiErr)
	}

	var out tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil && !errors.Is(err, io.EOF) {
		return "", fmt.Errorf("zoom: decode token response: %w", err)
	}
	if out.AccessToken == "" {
		return "", errors.New("zoom: token endpoint returned empty access_token")
	}
	if out.ExpiresIn <= 0 {
		return "", fmt.Errorf("zoom: token endpoint returned non-positive expires_in=%d", out.ExpiresIn)
	}

	ts.token = out.AccessToken
	ts.expiresAt = time.Now().Add(time.Duration(out.ExpiresIn) * time.Second)
	if strings.TrimSpace(out.APIURL) != "" {
		ts.apiURL = strings.TrimRight(out.APIURL, "/")
	}
	return ts.token, nil
}

func basicAuth(clientID, clientSecret string) string {
	return base64.StdEncoding.EncodeToString([]byte(clientID + ":" + clientSecret))
}
