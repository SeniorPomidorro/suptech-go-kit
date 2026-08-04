package zoom

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/SeniorPomidorro/suptech-go-kit/pkg/transport"
)

// Option configures the Zoom client.
type Option func(*config) error

type config struct {
	transport    *transport.Client
	oauthURL     string
	apiURL       string
	accountID    string
	clientID     string
	clientSecret string
}

// Client is a Zoom REST API client backed by Server-to-Server OAuth.
type Client struct {
	transport *transport.Client
	tokens    *tokenSource

	// staticAPIURL is set via WithAPIURL and overrides the api_url learned from the token response.
	// Useful for tests against httptest.Server. Empty means "use api_url from token".
	staticAPIURL string

	meetings   *MeetingsService
	recordings *RecordingsService
	users      *UsersService
}

// NewClient creates a Zoom client. WithS2SAuth is required.
func NewClient(opts ...Option) (*Client, error) {
	cfg := config{}
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		if err := opt(&cfg); err != nil {
			return nil, err
		}
	}

	if cfg.transport == nil {
		cfg.transport = transport.New()
	}

	ts, err := newTokenSource(cfg.transport, cfg.oauthURL, cfg.accountID, cfg.clientID, cfg.clientSecret)
	if err != nil {
		return nil, err
	}

	c := &Client{
		transport:    cfg.transport,
		tokens:       ts,
		staticAPIURL: strings.TrimRight(strings.TrimSpace(cfg.apiURL), "/"),
	}
	c.meetings = &MeetingsService{client: c}
	c.recordings = &RecordingsService{client: c}
	c.users = &UsersService{client: c}
	return c, nil
}

// WithS2SAuth configures Server-to-Server OAuth credentials.
func WithS2SAuth(accountID, clientID, clientSecret string) Option {
	return func(cfg *config) error {
		cfg.accountID = strings.TrimSpace(accountID)
		cfg.clientID = strings.TrimSpace(clientID)
		cfg.clientSecret = strings.TrimSpace(clientSecret)
		return nil
	}
}

// WithTransport injects a shared transport.Client (retries, timeouts, logger).
func WithTransport(tr *transport.Client) Option {
	return func(cfg *config) error {
		cfg.transport = tr
		return nil
	}
}

// WithOAuthURL overrides the token endpoint. Defaults to DefaultOAuthURL.
func WithOAuthURL(rawURL string) Option {
	return func(cfg *config) error {
		cfg.oauthURL = strings.TrimSpace(rawURL)
		return nil
	}
}

// WithAPIURL pins the API base URL, overriding the api_url field returned by the
// token endpoint. Mainly for tests; in production prefer the cluster URL Zoom returns.
func WithAPIURL(rawURL string) Option {
	return func(cfg *config) error {
		cfg.apiURL = rawURL
		return nil
	}
}

// Meetings returns the meetings API service.
func (c *Client) Meetings() *MeetingsService { return c.meetings }

// Recordings returns the cloud recordings API service.
func (c *Client) Recordings() *RecordingsService { return c.recordings }

// Users returns the users API service.
func (c *Client) Users() *UsersService { return c.users }

// apiBaseURL resolves the base URL for API calls.
// Priority: explicit override (WithAPIURL) > api_url from token response > DefaultAPIURL.
func (c *Client) apiBaseURL() string {
	if c.staticAPIURL != "" {
		return c.staticAPIURL
	}
	if u := c.tokens.APIURL(); u != "" {
		return u
	}
	return DefaultAPIURL
}

// doJSON executes an authenticated request, transparently refreshing the token
// once on 401 with code 124 (invalid access token).
func (c *Client) doJSON(ctx context.Context, method, path string, query url.Values, body, out any) error {
	err := c.send(ctx, method, path, query, body, out)
	if err == nil || !IsInvalidToken(err) {
		return err
	}
	c.tokens.Invalidate()
	return c.send(ctx, method, path, query, body, out)
}

func (c *Client) send(ctx context.Context, method, path string, query url.Values, body, out any) error {
	// Ensure we have a token (and api_url) before computing the request URL.
	token, err := c.tokens.Token(ctx)
	if err != nil {
		return err
	}

	req, err := c.buildRequest(ctx, method, path, query, body)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.transport.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return asZoomError(transport.NewAPIError(resp, 0))
	}

	if out == nil || resp.StatusCode == http.StatusNoContent {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("zoom: decode response: %w", err)
	}
	return nil
}

func (c *Client) buildRequest(ctx context.Context, method, path string, query url.Values, body any) (*http.Request, error) {
	base, err := url.Parse(c.apiBaseURL())
	if err != nil {
		return nil, fmt.Errorf("zoom: parse base URL: %w", err)
	}

	rel := path
	if !strings.HasPrefix(rel, "/") {
		rel = "/" + rel
	}
	// path is taken as already-escaped: RawPath preserves caller escaping (e.g. %2F inside a
	// meeting-UUID segment) instead of re-escaping the percent signs; for the common case of a
	// path with nothing to unescape, Path == RawPath and behavior is unchanged.
	unescaped, err := url.PathUnescape(rel)
	if err != nil {
		unescaped = rel
	}
	endpoint := *base
	endpoint.Path = strings.TrimRight(base.Path, "/") + unescaped
	endpoint.RawPath = strings.TrimRight(base.EscapedPath(), "/") + rel
	if len(query) > 0 {
		endpoint.RawQuery = query.Encode()
	}

	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("zoom: marshal body: %w", err)
		}
		reader = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint.String(), reader)
	if err != nil {
		return nil, fmt.Errorf("zoom: build request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	return req, nil
}
