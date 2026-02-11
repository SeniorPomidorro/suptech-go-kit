package atlassian

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/SeniorPomidorro/suptech-go-kit/pkg/transport"
)

// AuthMode defines Jira authentication variant.
type AuthMode string

const (
	AuthBasicEmailToken AuthMode = "basic_email_token"
	AuthBearerToken     AuthMode = "bearer"
	AuthBasicToken      AuthMode = "basic_token"
)

// Auth defines Jira credentials.
type Auth struct {
	Mode  AuthMode
	Email string
	Token string
}

// Option configures Atlassian client.
type Option func(*config) error

type config struct {
	baseURL           string
	auth              Auth
	transport         *transport.Client
	assetsCloudID     string
	assetsWorkspaceID string
	opsCloudID        string
}

// Client is Atlassian/Jira HTTP API client.
type Client struct {
	baseURL           *url.URL
	auth              Auth
	transport         *transport.Client
	assetsCloudID     string
	assetsWorkspaceID string
	opsCloudID        string

	issues     *IssuesService
	users      *UsersService
	assets     *AssetsService
	operations *OperationsService
}

// NewClient creates Atlassian client.
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

	if strings.TrimSpace(cfg.baseURL) == "" {
		return nil, errors.New("atlassian: base URL is required")
	}

	parsedURL, err := url.Parse(cfg.baseURL)
	if err != nil {
		return nil, fmt.Errorf("atlassian: parse base URL: %w", err)
	}
	if parsedURL.Scheme == "" || parsedURL.Host == "" {
		return nil, errors.New("atlassian: base URL must include scheme and host")
	}

	if cfg.transport == nil {
		cfg.transport = transport.New()
	}

	client := &Client{
		baseURL:           parsedURL,
		auth:              cfg.auth,
		transport:         cfg.transport,
		assetsCloudID:     cfg.assetsCloudID,
		assetsWorkspaceID: cfg.assetsWorkspaceID,
		opsCloudID:        strings.TrimSpace(cfg.opsCloudID),
	}
	if client.opsCloudID == "" {
		client.opsCloudID = strings.TrimSpace(cfg.assetsCloudID)
	}
	client.issues = &IssuesService{client: client}
	client.users = &UsersService{client: client}
	client.assets = &AssetsService{client: client}
	client.operations = &OperationsService{client: client}

	return client, nil
}

// WithBaseURL sets Jira base URL.
func WithBaseURL(baseURL string) Option {
	return func(cfg *config) error {
		cfg.baseURL = baseURL
		return nil
	}
}

// WithAuth sets authentication mode and credentials.
func WithAuth(auth Auth) Option {
	return func(cfg *config) error {
		cfg.auth = auth
		return nil
	}
}

// WithTransport injects shared transport.
func WithTransport(tr *transport.Client) Option {
	return func(cfg *config) error {
		cfg.transport = tr
		return nil
	}
}

// WithAssetsCloudID configures Jira Assets cloud ID.
func WithAssetsCloudID(cloudID string) Option {
	return func(cfg *config) error {
		cfg.assetsCloudID = strings.TrimSpace(cloudID)
		return nil
	}
}

// WithAssetsWorkspaceID configures Jira Assets workspace ID.
func WithAssetsWorkspaceID(workspaceID string) Option {
	return func(cfg *config) error {
		cfg.assetsWorkspaceID = strings.TrimSpace(workspaceID)
		return nil
	}
}

// WithOpsCloudID configures Jira Operations API cloud ID.
func WithOpsCloudID(cloudID string) Option {
	return func(cfg *config) error {
		cfg.opsCloudID = strings.TrimSpace(cloudID)
		return nil
	}
}

// Issues returns issues API service.
func (c *Client) Issues() *IssuesService {
	return c.issues
}

// Users returns users API service.
func (c *Client) Users() *UsersService {
	return c.users
}

// Assets returns Jira Assets API service.
func (c *Client) Assets() *AssetsService {
	return c.assets
}

// Operations returns Jira Operations API service.
func (c *Client) Operations() *OperationsService {
	return c.operations
}

func (c *Client) newRequest(ctx context.Context, method, path string, query url.Values, body any) (*http.Request, error) {
	var payload []byte
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("atlassian: marshal request body: %w", err)
		}
		payload = encoded
	}

	return c.newRawRequest(ctx, method, path, query, payload, "application/json")
}

func (c *Client) newRawRequest(ctx context.Context, method, path string, query url.Values, body []byte, contentType string) (*http.Request, error) {
	if c == nil {
		return nil, errors.New("atlassian: client is nil")
	}

	rel := path
	if !strings.HasPrefix(rel, "/") {
		rel = "/" + rel
	}

	endpoint := c.baseURL.ResolveReference(&url.URL{Path: rel})
	endpoint.RawQuery = query.Encode()

	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint.String(), reader)
	if err != nil {
		return nil, fmt.Errorf("atlassian: create request: %w", err)
	}

	if body != nil && contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	req.Header.Set("Accept", "application/json")

	authValue, err := c.authHeaderValue()
	if err != nil {
		return nil, err
	}
	if authValue != "" {
		req.Header.Set("Authorization", authValue)
	}

	return req, nil
}

func (c *Client) authHeaderValue() (string, error) {
	switch c.auth.Mode {
	case "":
		return "", nil
	case AuthBasicEmailToken:
		if strings.TrimSpace(c.auth.Email) == "" || strings.TrimSpace(c.auth.Token) == "" {
			return "", errors.New("atlassian: email and token are required for basic email/token auth")
		}
		raw := c.auth.Email + ":" + c.auth.Token
		return "Basic " + base64.StdEncoding.EncodeToString([]byte(raw)), nil
	case AuthBearerToken:
		if strings.TrimSpace(c.auth.Token) == "" {
			return "", errors.New("atlassian: token is required for bearer auth")
		}
		return "Bearer " + c.auth.Token, nil
	case AuthBasicToken:
		if strings.TrimSpace(c.auth.Token) == "" {
			return "", errors.New("atlassian: token is required for direct basic auth")
		}
		return "Basic " + c.auth.Token, nil
	default:
		return "", fmt.Errorf("atlassian: unsupported auth mode %q", c.auth.Mode)
	}
}

func (c *Client) doNoResponseBody(req *http.Request) error {
	resp, err := c.transport.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return transport.NewAPIError(resp, 0)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	return nil
}
