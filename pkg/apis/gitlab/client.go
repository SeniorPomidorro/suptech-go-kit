package gitlab

import (
	"github.com/SeniorPomidorro/suptech-go-kit/pkg/transport"
)

// Option configures GitLab client.
type Option func(*config)

type config struct {
	token     string
	transport *transport.Client
}

// Client is a minimal GitLab API client.
type Client struct {
	token     string
	transport *transport.Client
}

// NewClient creates GitLab API client.
func NewClient(opts ...Option) *Client {
	cfg := config{}
	for _, opt := range opts {
		if opt != nil {
			opt(&cfg)
		}
	}
	if cfg.transport == nil {
		cfg.transport = transport.New()
	}
	return &Client{
		token:     cfg.token,
		transport: cfg.transport,
	}
}

// WithToken sets PRIVATE-TOKEN value.
func WithToken(token string) Option {
	return func(cfg *config) {
		cfg.token = token
	}
}

// WithTransport injects shared HTTP transport.
func WithTransport(tr *transport.Client) Option {
	return func(cfg *config) {
		if tr != nil {
			cfg.transport = tr
		}
	}
}
