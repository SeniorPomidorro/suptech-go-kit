package slack

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

const defaultBaseURL = "https://slack.com/api"

// Option configures Slack client.
type Option func(*config)

type config struct {
	baseURL   string
	token     string
	teamID    string
	transport *transport.Client
}

// Client is Slack Web API client.
type Client struct {
	baseURL   *url.URL
	token     string
	teamID    string
	transport *transport.Client

	userGroups    *UserGroupsService
	conversations *ConversationsService
	messages      *MessagesService
	users         *UsersService
	views         *ViewsService
	canvas        *CanvasService
}

// NewClient creates Slack Web API client.
func NewClient(opts ...Option) (*Client, error) {
	cfg := config{
		baseURL: defaultBaseURL,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}

	parsedBaseURL, err := url.Parse(cfg.baseURL)
	if err != nil {
		return nil, fmt.Errorf("slack: parse base URL: %w", err)
	}
	if parsedBaseURL.Scheme == "" || parsedBaseURL.Host == "" {
		return nil, errors.New("slack: base URL must include scheme and host")
	}
	if cfg.transport == nil {
		cfg.transport = transport.New()
	}

	client := &Client{
		baseURL:   parsedBaseURL,
		token:     strings.TrimSpace(cfg.token),
		teamID:    strings.TrimSpace(cfg.teamID),
		transport: cfg.transport,
	}
	client.userGroups = &UserGroupsService{client: client}
	client.conversations = &ConversationsService{client: client}
	client.messages = &MessagesService{client: client}
	client.users = &UsersService{client: client}
	client.views = &ViewsService{client: client}
	client.canvas = &CanvasService{client: client}

	return client, nil
}

// WithBaseURL overrides Slack API base URL.
func WithBaseURL(baseURL string) Option {
	return func(cfg *config) {
		cfg.baseURL = baseURL
	}
}

// WithToken sets Bot/User token used in Authorization header.
func WithToken(token string) Option {
	return func(cfg *config) {
		cfg.token = token
	}
}

// WithTeamID sets optional team_id for grid/org tokens.
func WithTeamID(teamID string) Option {
	return func(cfg *config) {
		cfg.teamID = teamID
	}
}

// WithTransport injects shared transport client.
func WithTransport(tr *transport.Client) Option {
	return func(cfg *config) {
		if tr != nil {
			cfg.transport = tr
		}
	}
}

// UserGroups returns user groups API service.
func (c *Client) UserGroups() *UserGroupsService {
	return c.userGroups
}

// Conversations returns conversations API service.
func (c *Client) Conversations() *ConversationsService {
	return c.conversations
}

// Messages returns messages API service.
func (c *Client) Messages() *MessagesService {
	return c.messages
}

// Users returns users API service.
func (c *Client) Users() *UsersService {
	return c.users
}

// Views returns views API service.
func (c *Client) Views() *ViewsService {
	return c.views
}

// Canvas returns canvas API service.
func (c *Client) Canvas() *CanvasService {
	return c.canvas
}

func (c *Client) newFormRequest(ctx context.Context, method string, form url.Values) (*http.Request, error) {
	if form == nil {
		form = url.Values{}
	}
	body := form.Encode()

	req, err := c.newRawRequest(ctx, method, strings.NewReader(body), "application/x-www-form-urlencoded")
	if err != nil {
		return nil, err
	}
	return req, nil
}

func (c *Client) newJSONRequest(ctx context.Context, method string, payload any) (*http.Request, error) {
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("slack: marshal request body: %w", err)
	}
	return c.newRawRequest(ctx, method, bytes.NewReader(data), "application/json")
}

func (c *Client) newRawRequest(ctx context.Context, method string, body io.Reader, contentType string) (*http.Request, error) {
	endpoint, err := resolveSlackMethodURL(c.baseURL, method)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), body)
	if err != nil {
		return nil, fmt.Errorf("slack: create request: %w", err)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	req.Header.Set("Accept", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	return req, nil
}

func resolveSlackMethodURL(base *url.URL, method string) (*url.URL, error) {
	if base == nil {
		return nil, errors.New("slack: base URL is not configured")
	}
	methodPath := strings.TrimSpace(method)
	if methodPath == "" {
		return nil, errors.New("slack: method path is required")
	}

	endpoint := *base
	basePath := strings.TrimRight(endpoint.Path, "/")
	methodPath = strings.TrimLeft(methodPath, "/")
	if basePath == "" {
		endpoint.Path = "/" + methodPath
	} else {
		endpoint.Path = basePath + "/" + methodPath
	}
	endpoint.RawQuery = ""
	endpoint.Fragment = ""
	return &endpoint, nil
}

func (c *Client) do(req *http.Request, out any) error {
	resp, err := c.transport.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return transport.NewAPIError(resp, 0)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("slack: read response body: %w", err)
	}
	if len(body) == 0 {
		return nil
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err == nil {
		if okRaw, hasOK := raw["ok"]; hasOK {
			var ok bool
			if err := json.Unmarshal(okRaw, &ok); err == nil && !ok {
				return parseSlackError(raw)
			}
		}
	}

	if out == nil {
		return nil
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("slack: decode response: %w", err)
	}
	return nil
}

func parseSlackError(raw map[string]json.RawMessage) error {
	result := &Error{
		Code:     rawString(raw, "error"),
		Needed:   rawString(raw, "needed"),
		Provided: rawString(raw, "provided"),
		Warning:  rawString(raw, "warning"),
	}
	return result
}

func rawString(raw map[string]json.RawMessage, key string) string {
	value, ok := raw[key]
	if !ok {
		return ""
	}
	var result string
	if err := json.Unmarshal(value, &result); err != nil {
		return ""
	}
	return result
}

func (c *Client) withTeamID(form url.Values) {
	if form == nil {
		return
	}
	if c.teamID != "" && form.Get("team_id") == "" {
		form.Set("team_id", c.teamID)
	}
}
