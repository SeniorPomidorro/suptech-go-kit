package transport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"sync"
	"time"
)

const defaultErrorBodyLimit int64 = 4096

// Logger describes the minimal logging API used by the transport client.
type Logger interface {
	Printf(format string, args ...any)
}

// RetryConfig controls retry behavior for transient failures.
type RetryConfig struct {
	MaxAttempts    int
	InitialBackoff time.Duration
	MaxBackoff     time.Duration
	Jitter         time.Duration
}

var defaultRetryConfig = RetryConfig{
	MaxAttempts:    3,
	InitialBackoff: 200 * time.Millisecond,
	MaxBackoff:     2 * time.Second,
	Jitter:         100 * time.Millisecond,
}

// Client is a shared HTTP layer for all API packages.
type Client struct {
	httpClient     *http.Client
	retry          RetryConfig
	logger         Logger
	baseHeaders    http.Header
	errorBodyLimit int64

	randMu sync.Mutex
	rand   *rand.Rand
}

// Option mutates Client behavior.
type Option func(*Client)

// New creates a transport client with sane defaults.
func New(opts ...Option) *Client {
	c := &Client{
		httpClient:     &http.Client{Timeout: 30 * time.Second},
		retry:          normalizeRetryConfig(defaultRetryConfig),
		baseHeaders:    http.Header{},
		errorBodyLimit: defaultErrorBodyLimit,
		rand:           rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	for _, opt := range opts {
		if opt != nil {
			opt(c)
		}
	}

	c.retry = normalizeRetryConfig(c.retry)
	if c.baseHeaders == nil {
		c.baseHeaders = http.Header{}
	}
	if c.httpClient == nil {
		c.httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	if c.errorBodyLimit <= 0 {
		c.errorBodyLimit = defaultErrorBodyLimit
	}

	return c
}

// WithHTTPClient injects custom HTTP client instance.
func WithHTTPClient(client *http.Client) Option {
	return func(c *Client) {
		if client != nil {
			c.httpClient = client
		}
	}
}

// WithTimeout sets a client-level timeout.
func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) {
		if timeout <= 0 {
			return
		}
		if c.httpClient == nil {
			c.httpClient = &http.Client{}
		}
		c.httpClient.Timeout = timeout
	}
}

// WithRetry overrides retry policy.
func WithRetry(cfg RetryConfig) Option {
	return func(c *Client) {
		c.retry = normalizeRetryConfig(cfg)
	}
}

// WithLogger configures request logging.
func WithLogger(logger Logger) Option {
	return func(c *Client) {
		c.logger = logger
	}
}

// WithBaseHeaders applies headers to every request unless already present.
func WithBaseHeaders(headers http.Header) Option {
	return func(c *Client) {
		if len(headers) == 0 {
			return
		}
		if c.baseHeaders == nil {
			c.baseHeaders = http.Header{}
		}
		for key, values := range headers {
			for _, value := range values {
				c.baseHeaders.Add(key, value)
			}
		}
	}
}

// WithErrorBodyLimit changes max amount of response body captured in APIError.
func WithErrorBodyLimit(limit int64) Option {
	return func(c *Client) {
		if limit > 0 {
			c.errorBodyLimit = limit
		}
	}
}

// Do executes request with retries for transient failures.
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	if req == nil {
		return nil, errors.New("transport: request is nil")
	}

	attempts := c.retry.MaxAttempts
	replayable := req.Body == nil || req.GetBody != nil
	if !replayable {
		attempts = 1
	}

	var lastErr error
	for attempt := 1; attempt <= attempts; attempt++ {
		attemptReq, err := c.requestForAttempt(req, attempt)
		if err != nil {
			return nil, err
		}
		c.applyBaseHeaders(attemptReq.Header)

		resp, err := c.httpClient.Do(attemptReq)
		if err != nil {
			if !shouldRetryError(err) || attempt == attempts {
				return nil, err
			}
			lastErr = err
			if sleepErr := sleepWithContext(req.Context(), c.nextBackoff(attempt, 0)); sleepErr != nil {
				return nil, sleepErr
			}
			continue
		}

		if shouldRetryStatus(resp.StatusCode) && attempt < attempts {
			drainAndClose(resp.Body)
			if sleepErr := sleepWithContext(req.Context(), c.nextBackoff(attempt, parseRetryAfter(resp.Header.Get("Retry-After")))); sleepErr != nil {
				return nil, sleepErr
			}
			continue
		}

		if c.logger != nil {
			c.logger.Printf("transport: %s %s -> %d (attempt=%d)", req.Method, req.URL.Redacted(), resp.StatusCode, attempt)
		}
		return resp, nil
	}

	if lastErr != nil {
		return nil, lastErr
	}
	return nil, errors.New("transport: request failed after retries")
}

// DoJSON executes request and decodes JSON body for successful responses.
func (c *Client) DoJSON(req *http.Request, out any) error {
	if req == nil {
		return errors.New("transport: request is nil")
	}
	if req.Header.Get("Accept") == "" {
		req.Header.Set("Accept", "application/json")
	}

	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return NewAPIError(resp, c.errorBodyLimit)
	}

	if out == nil || resp.StatusCode == http.StatusNoContent {
		drainAndClose(resp.Body)
		return nil
	}

	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(out); err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}
		return fmt.Errorf("transport: decode response: %w", err)
	}

	return nil
}

func (c *Client) requestForAttempt(req *http.Request, attempt int) (*http.Request, error) {
	clone := req.Clone(req.Context())
	clone.Header = req.Header.Clone()

	if req.Body == nil {
		return clone, nil
	}
	if attempt == 1 {
		clone.Body = req.Body
		return clone, nil
	}
	if req.GetBody == nil {
		return nil, errors.New("transport: request body is not replayable")
	}

	body, err := req.GetBody()
	if err != nil {
		return nil, fmt.Errorf("transport: clone body for retry: %w", err)
	}
	clone.Body = body
	return clone, nil
}

func (c *Client) applyBaseHeaders(headers http.Header) {
	for key, values := range c.baseHeaders {
		if headers.Get(key) != "" {
			continue
		}
		for _, value := range values {
			headers.Add(key, value)
		}
	}
}

func (c *Client) nextBackoff(attempt int, retryAfter time.Duration) time.Duration {
	if retryAfter > 0 {
		return retryAfter
	}

	backoff := c.retry.InitialBackoff
	for i := 1; i < attempt; i++ {
		backoff *= 2
		if backoff >= c.retry.MaxBackoff {
			backoff = c.retry.MaxBackoff
			break
		}
	}

	if c.retry.Jitter > 0 {
		c.randMu.Lock()
		backoff += time.Duration(c.rand.Int63n(int64(c.retry.Jitter)))
		c.randMu.Unlock()
	}

	if backoff > c.retry.MaxBackoff {
		backoff = c.retry.MaxBackoff
	}

	return backoff
}

func normalizeRetryConfig(cfg RetryConfig) RetryConfig {
	if cfg.MaxAttempts <= 0 {
		cfg.MaxAttempts = defaultRetryConfig.MaxAttempts
	}
	if cfg.InitialBackoff <= 0 {
		cfg.InitialBackoff = defaultRetryConfig.InitialBackoff
	}
	if cfg.MaxBackoff <= 0 {
		cfg.MaxBackoff = defaultRetryConfig.MaxBackoff
	}
	if cfg.MaxBackoff < cfg.InitialBackoff {
		cfg.MaxBackoff = cfg.InitialBackoff
	}
	if cfg.Jitter < 0 {
		cfg.Jitter = 0
	}
	return cfg
}

func shouldRetryError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	var netErr net.Error
	return errors.As(err, &netErr)
}

func shouldRetryStatus(statusCode int) bool {
	if statusCode == http.StatusTooManyRequests {
		return true
	}
	return statusCode == http.StatusInternalServerError ||
		statusCode == http.StatusBadGateway ||
		statusCode == http.StatusServiceUnavailable ||
		statusCode == http.StatusGatewayTimeout
}

func parseRetryAfter(raw string) time.Duration {
	if raw == "" {
		return 0
	}

	if seconds, err := time.ParseDuration(raw + "s"); err == nil && seconds > 0 {
		return seconds
	}

	if at, err := http.ParseTime(raw); err == nil {
		delay := time.Until(at)
		if delay > 0 {
			return delay
		}
	}

	return 0
}

func sleepWithContext(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func drainAndClose(body io.ReadCloser) {
	if body == nil {
		return
	}
	_, _ = io.Copy(io.Discard, body)
	_ = body.Close()
}
