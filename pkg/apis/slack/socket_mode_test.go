package slack

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/SeniorPomidorro/suptech-go-kit/pkg/transport"
)

func TestSocketModeRunWithHandlerAcksEnvelope(t *testing.T) {
	t.Parallel()

	var authHeader string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader = r.Header.Get("Authorization")
		if r.URL.Path != "/api/apps.connections.open" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"url":"ws://socket.example/connection-1"}`))
	}))
	defer srv.Close()

	conn := &fakeSocketModeConn{
		readMessages: []string{
			`{"type":"slash_commands","envelope_id":"env-1","accepts_response_payload":true,"payload":{"command":"/deploy"}}`,
		},
	}
	dialer := &fakeSocketModeDialer{
		conns: []SocketModeConn{conn},
	}

	client := NewSocketModeClient(
		WithAppLevelToken("xapp-test"),
		WithSocketModeBaseURL(srv.URL+"/api"),
		WithSocketModeTransport(transport.New()),
		WithSocketModeDialer(dialer),
		WithSocketModeReconnectDelay(0),
	)

	ctx, cancel := context.WithCancel(context.Background())
	err := client.RunWithHandler(ctx, SocketModeHandlerFunc(func(ctx context.Context, event SocketModeEvent) (*SocketModeResponse, error) {
		cancel()
		return &SocketModeResponse{Payload: map[string]any{"text": "ack"}}, nil
	}))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}

	if authHeader != "Bearer xapp-test" {
		t.Fatalf("unexpected Authorization header: %q", authHeader)
	}
	if len(dialer.wsURLs) != 1 || dialer.wsURLs[0] != "ws://socket.example/connection-1" {
		t.Fatalf("unexpected dial calls: %+v", dialer.wsURLs)
	}

	writes := conn.writesSnapshot()
	if len(writes) != 1 {
		t.Fatalf("expected one ACK frame, got %d", len(writes))
	}
	if writes[0]["envelope_id"] != "env-1" {
		t.Fatalf("unexpected envelope_id: %+v", writes[0]["envelope_id"])
	}
	payload, ok := writes[0]["payload"].(map[string]any)
	if !ok {
		t.Fatalf("expected payload in ACK, got %+v", writes[0])
	}
	if payload["text"] != "ack" {
		t.Fatalf("unexpected payload: %+v", payload)
	}
}

func TestSocketModeRunWithHandlerErrorStillAcks(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"url":"ws://socket.example/connection-2"}`))
	}))
	defer srv.Close()

	conn := &fakeSocketModeConn{
		readMessages: []string{
			`{"type":"events_api","envelope_id":"env-2","accepts_response_payload":true,"payload":{"type":"event_callback"}}`,
		},
	}
	dialer := &fakeSocketModeDialer{
		conns: []SocketModeConn{conn},
	}

	client := NewSocketModeClient(
		WithAppLevelToken("xapp-test"),
		WithSocketModeBaseURL(srv.URL),
		WithSocketModeTransport(transport.New()),
		WithSocketModeDialer(dialer),
		WithSocketModeReconnectDelay(0),
	)

	ctx, cancel := context.WithCancel(context.Background())
	err := client.RunWithHandler(ctx, SocketModeHandlerFunc(func(ctx context.Context, event SocketModeEvent) (*SocketModeResponse, error) {
		cancel()
		return nil, errors.New("handler failed")
	}))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}

	writes := conn.writesSnapshot()
	if len(writes) != 1 {
		t.Fatalf("expected one ACK frame, got %d", len(writes))
	}
	if writes[0]["envelope_id"] != "env-2" {
		t.Fatalf("unexpected envelope_id: %+v", writes[0]["envelope_id"])
	}
	if _, hasPayload := writes[0]["payload"]; hasPayload {
		t.Fatalf("did not expect payload on handler error: %+v", writes[0])
	}
}

func TestSocketModeRunWithHandlerHelloNoAck(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"url":"ws://socket.example/connection-3"}`))
	}))
	defer srv.Close()

	conn := &fakeSocketModeConn{
		readMessages: []string{
			`{"type":"hello","payload":{"num_connections":1}}`,
		},
	}
	dialer := &fakeSocketModeDialer{
		conns: []SocketModeConn{conn},
	}

	client := NewSocketModeClient(
		WithAppLevelToken("xapp-test"),
		WithSocketModeBaseURL(srv.URL),
		WithSocketModeTransport(transport.New()),
		WithSocketModeDialer(dialer),
		WithSocketModeReconnectDelay(0),
	)

	ctx, cancel := context.WithCancel(context.Background())
	err := client.RunWithHandler(ctx, SocketModeHandlerFunc(func(ctx context.Context, event SocketModeEvent) (*SocketModeResponse, error) {
		cancel()
		return nil, nil
	}))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}

	writes := conn.writesSnapshot()
	if len(writes) != 0 {
		t.Fatalf("did not expect ACK for hello event, got %d writes", len(writes))
	}
}

func TestSocketModeRunReconnectsAfterDialError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"url":"ws://socket.example/reconnect"}`))
	}))
	defer srv.Close()

	conn := &fakeSocketModeConn{
		readMessages: []string{
			`{"type":"events_api","envelope_id":"env-r","payload":{"ok":true}}`,
		},
	}
	dialer := &fakeSocketModeDialer{
		errs:  []error{errors.New("temporary dial failure")},
		conns: []SocketModeConn{conn},
	}

	client := NewSocketModeClient(
		WithAppLevelToken("xapp-test"),
		WithSocketModeBaseURL(srv.URL),
		WithSocketModeTransport(transport.New()),
		WithSocketModeDialer(dialer),
		WithSocketModeReconnectDelay(0),
	)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := client.RunWithHandler(ctx, SocketModeHandlerFunc(func(ctx context.Context, event SocketModeEvent) (*SocketModeResponse, error) {
		cancel()
		return nil, nil
	}))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}

	if len(dialer.wsURLs) < 2 {
		t.Fatalf("expected reconnect and second dial, got %d dials", len(dialer.wsURLs))
	}
}

func TestSocketModeRunValidationErrors(t *testing.T) {
	t.Parallel()

	client := NewSocketModeClient(
		WithSocketModeBaseURL("https://slack.com/api"),
		WithSocketModeTransport(transport.New()),
	)
	err := client.Run(context.Background())
	if err == nil || !strings.Contains(err.Error(), "app-level token is required") {
		t.Fatalf("expected missing token error, got %v", err)
	}

	client = NewSocketModeClient(
		WithAppLevelToken("xapp-test"),
		WithSocketModeBaseURL("://bad"),
		WithSocketModeTransport(transport.New()),
	)
	err = client.Run(context.Background())
	if err == nil || !strings.Contains(err.Error(), "base URL is invalid") {
		t.Fatalf("expected invalid base URL error, got %v", err)
	}
}

func TestSocketModeOpenConnectionSlackError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":false,"error":"invalid_auth"}`))
	}))
	defer srv.Close()

	client := NewSocketModeClient(
		WithAppLevelToken("xapp-test"),
		WithSocketModeBaseURL(srv.URL),
		WithSocketModeTransport(transport.New()),
		WithSocketModeDialer(&fakeSocketModeDialer{}),
	)
	err := client.Run(context.Background())
	var slackErr *Error
	if !errors.As(err, &slackErr) {
		t.Fatalf("expected slack.Error, got %T (%v)", err, err)
	}
	if slackErr.Code != "invalid_auth" {
		t.Fatalf("unexpected error code: %q", slackErr.Code)
	}
}

type fakeSocketModeDialer struct {
	mu sync.Mutex

	wsURLs []string
	conns  []SocketModeConn
	errs   []error
}

func (d *fakeSocketModeDialer) Dial(ctx context.Context, wsURL string) (SocketModeConn, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.wsURLs = append(d.wsURLs, wsURL)

	if len(d.errs) > 0 {
		err := d.errs[0]
		d.errs = d.errs[1:]
		return nil, err
	}
	if len(d.conns) == 0 {
		return nil, io.EOF
	}
	conn := d.conns[0]
	d.conns = d.conns[1:]
	return conn, nil
}

type fakeSocketModeConn struct {
	mu sync.Mutex

	readMessages []string
	readIndex    int
	writes       []map[string]any
	closed       bool
}

func (c *fakeSocketModeConn) ReadJSON(v any) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.readIndex >= len(c.readMessages) {
		return io.EOF
	}
	message := c.readMessages[c.readIndex]
	c.readIndex++
	return json.Unmarshal([]byte(message), v)
}

func (c *fakeSocketModeConn) WriteJSON(v any) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	payloadBytes, err := json.Marshal(v)
	if err != nil {
		return err
	}
	var payload map[string]any
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return err
	}
	c.writes = append(c.writes, payload)
	return nil
}

func (c *fakeSocketModeConn) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closed = true
	return nil
}

func (c *fakeSocketModeConn) writesSnapshot() []map[string]any {
	c.mu.Lock()
	defer c.mu.Unlock()
	result := make([]map[string]any, len(c.writes))
	copy(result, c.writes)
	return result
}
