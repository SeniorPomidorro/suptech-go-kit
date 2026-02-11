package slack

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/SeniorPomidorro/suptech-go-kit/pkg/transport"
)

const defaultSocketModeReconnectDelay = time.Second

// SocketModeEvent is a single event envelope delivered over socket mode.
type SocketModeEvent struct {
	Type                   string          `json:"type,omitempty"`
	EnvelopeID             string          `json:"envelope_id,omitempty"`
	AcceptsResponsePayload bool            `json:"accepts_response_payload,omitempty"`
	Payload                json.RawMessage `json:"payload,omitempty"`
	RetryAttempt           int             `json:"retry_attempt,omitempty"`
	RetryReason            string          `json:"retry_reason,omitempty"`
}

// SocketModeResponse contains optional payload sent in envelope ACK.
type SocketModeResponse struct {
	Payload any `json:"payload,omitempty"`
}

// SocketModeHandler processes socket mode events.
type SocketModeHandler interface {
	HandleEvent(ctx context.Context, event SocketModeEvent) (*SocketModeResponse, error)
}

// SocketModeHandlerFunc adapts function to SocketModeHandler.
type SocketModeHandlerFunc func(ctx context.Context, event SocketModeEvent) (*SocketModeResponse, error)

// HandleEvent calls f(ctx, event).
func (f SocketModeHandlerFunc) HandleEvent(ctx context.Context, event SocketModeEvent) (*SocketModeResponse, error) {
	return f(ctx, event)
}

// SocketModeConn abstracts websocket connection used by socket mode client.
type SocketModeConn interface {
	ReadJSON(v any) error
	WriteJSON(v any) error
	Close() error
}

// SocketModeDialer opens websocket connection to provided URL.
type SocketModeDialer interface {
	Dial(ctx context.Context, wsURL string) (SocketModeConn, error)
}

// SocketModeOption configures SocketModeClient.
type SocketModeOption func(*socketModeConfig)

type socketModeConfig struct {
	appToken       string
	baseURL        string
	transport      *transport.Client
	dialer         SocketModeDialer
	reconnectDelay time.Duration
	logger         transport.Logger
}

// SocketModeClient manages Slack socket mode lifecycle.
type SocketModeClient struct {
	appToken       string
	baseURL        *url.URL
	transport      *transport.Client
	dialer         SocketModeDialer
	reconnectDelay time.Duration
	logger         transport.Logger
}

// NewSocketModeClient creates a socket mode client.
func NewSocketModeClient(opts ...SocketModeOption) *SocketModeClient {
	cfg := socketModeConfig{
		baseURL:        defaultBaseURL,
		reconnectDelay: defaultSocketModeReconnectDelay,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	if cfg.transport == nil {
		cfg.transport = transport.New()
	}
	if cfg.dialer == nil {
		cfg.dialer = &rfc6455Dialer{}
	}
	parsedBaseURL, err := url.Parse(cfg.baseURL)
	if err != nil || parsedBaseURL.Scheme == "" || parsedBaseURL.Host == "" {
		parsedBaseURL = nil
	}
	if cfg.reconnectDelay < 0 {
		cfg.reconnectDelay = defaultSocketModeReconnectDelay
	}

	return &SocketModeClient{
		appToken:       strings.TrimSpace(cfg.appToken),
		baseURL:        parsedBaseURL,
		transport:      cfg.transport,
		dialer:         cfg.dialer,
		reconnectDelay: cfg.reconnectDelay,
		logger:         cfg.logger,
	}
}

// WithAppLevelToken sets app-level token for socket mode.
func WithAppLevelToken(appToken string) SocketModeOption {
	return func(cfg *socketModeConfig) {
		cfg.appToken = appToken
	}
}

// WithSocketModeBaseURL overrides Slack API base URL for apps.connections.open.
func WithSocketModeBaseURL(baseURL string) SocketModeOption {
	return func(cfg *socketModeConfig) {
		cfg.baseURL = baseURL
	}
}

// WithSocketModeTransport injects transport used by apps.connections.open calls.
func WithSocketModeTransport(tr *transport.Client) SocketModeOption {
	return func(cfg *socketModeConfig) {
		if tr != nil {
			cfg.transport = tr
		}
	}
}

// WithSocketModeDialer overrides websocket dialer implementation.
func WithSocketModeDialer(dialer SocketModeDialer) SocketModeOption {
	return func(cfg *socketModeConfig) {
		if dialer != nil {
			cfg.dialer = dialer
		}
	}
}

// WithSocketModeReconnectDelay sets reconnect delay after connection errors.
func WithSocketModeReconnectDelay(delay time.Duration) SocketModeOption {
	return func(cfg *socketModeConfig) {
		cfg.reconnectDelay = delay
	}
}

// WithSocketModeLogger sets optional logger for socket mode runtime diagnostics.
func WithSocketModeLogger(logger transport.Logger) SocketModeOption {
	return func(cfg *socketModeConfig) {
		cfg.logger = logger
	}
}

// Run starts socket mode processing loop.
func (c *SocketModeClient) Run(ctx context.Context) error {
	return c.RunWithHandler(ctx, nil)
}

// RunWithHandler starts socket mode loop and dispatches events to handler.
func (c *SocketModeClient) RunWithHandler(ctx context.Context, handler SocketModeHandler) error {
	if ctx == nil {
		return errors.New("slack: context is required")
	}
	if strings.TrimSpace(c.appToken) == "" {
		return errors.New("slack: app-level token is required")
	}
	if c.baseURL == nil {
		return errors.New("slack: socket mode base URL is invalid")
	}
	if c.transport == nil {
		return errors.New("slack: socket mode transport is not configured")
	}
	if c.dialer == nil {
		return errors.New("slack: socket mode dialer is not configured")
	}

	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		socketURL, err := c.openConnection(ctx)
		if err != nil {
			return err
		}

		conn, err := c.dialer.Dial(ctx, socketURL)
		if err != nil {
			if c.logger != nil {
				c.logger.Printf("slack socket mode: dial failed: %v", err)
			}
			if waitErr := c.waitReconnect(ctx); waitErr != nil {
				return waitErr
			}
			continue
		}

		err = c.processConnection(ctx, conn, handler)
		_ = conn.Close()

		if err == nil {
			if waitErr := c.waitReconnect(ctx); waitErr != nil {
				return waitErr
			}
			continue
		}
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return err
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}

		if c.logger != nil {
			c.logger.Printf("slack socket mode: connection ended: %v", err)
		}
		if waitErr := c.waitReconnect(ctx); waitErr != nil {
			return waitErr
		}
	}
}

func (c *SocketModeClient) processConnection(ctx context.Context, conn SocketModeConn, handler SocketModeHandler) error {
	stop := make(chan struct{})
	defer close(stop)

	go func() {
		select {
		case <-ctx.Done():
			_ = conn.Close()
		case <-stop:
		}
	}()

	for {
		var event SocketModeEvent
		if err := conn.ReadJSON(&event); err != nil {
			return err
		}

		var response *SocketModeResponse
		if handler != nil {
			handlerResponse, err := handler.HandleEvent(ctx, event)
			if err != nil {
				if c.logger != nil {
					c.logger.Printf("slack socket mode: handler error: %v", err)
				}
			} else {
				response = handlerResponse
			}
		}

		if strings.TrimSpace(event.EnvelopeID) == "" {
			continue
		}

		ack := map[string]any{
			"envelope_id": event.EnvelopeID,
		}
		if response != nil && event.AcceptsResponsePayload {
			ack["payload"] = response.Payload
		}

		if err := conn.WriteJSON(ack); err != nil {
			return err
		}
	}
}

func (c *SocketModeClient) waitReconnect(ctx context.Context) error {
	if c.reconnectDelay <= 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			return nil
		}
	}

	timer := time.NewTimer(c.reconnectDelay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (c *SocketModeClient) openConnection(ctx context.Context) (string, error) {
	endpoint, err := resolveSlackMethodURL(c.baseURL, "apps.connections.open")
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), nil)
	if err != nil {
		return "", fmt.Errorf("slack: create apps.connections.open request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.appToken)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.transport.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return "", transport.NewAPIError(resp, 0)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("slack: read apps.connections.open response: %w", err)
	}
	if len(body) == 0 {
		return "", errors.New("slack: empty apps.connections.open response")
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return "", fmt.Errorf("slack: decode apps.connections.open response: %w", err)
	}

	if okRaw, hasOK := raw["ok"]; hasOK {
		var ok bool
		if err := json.Unmarshal(okRaw, &ok); err == nil && !ok {
			return "", parseSlackError(raw)
		}
	}

	socketURL := rawString(raw, "url")
	if strings.TrimSpace(socketURL) == "" {
		return "", errors.New("slack: apps.connections.open did not return socket URL")
	}
	return socketURL, nil
}
