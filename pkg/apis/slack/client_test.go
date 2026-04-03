package slack

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/SeniorPomidorro/suptech-go-kit/pkg/transport"
)

func TestPostMessageReturnsSlackErrorOnOKFalse(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat.postMessage" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer xoxb-test" {
			t.Fatalf("unexpected authorization header: %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":false,"error":"channel_not_found"}`))
	}))
	defer srv.Close()

	client, err := NewClient(
		WithBaseURL(srv.URL),
		WithToken("xoxb-test"),
		WithTransport(transport.New()),
	)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	_, err = client.Messages().PostMessage(context.Background(), &PostMessageRequest{Channel: "C123", Text: "hello"})
	if err == nil {
		t.Fatalf("expected error")
	}
	slackErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("expected slack.Error, got %T", err)
	}
	if slackErr.Code != "channel_not_found" {
		t.Fatalf("unexpected error code: %q", slackErr.Code)
	}
}

func TestGetConversationListUsesCursorPagination(t *testing.T) {
	t.Parallel()

	requestCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/conversations.list" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer xoxb-test" {
			t.Fatalf("unexpected authorization header: %q", got)
		}

		q := r.URL.Query()
		if q.Get("types") != "public_channel,private_channel" {
			t.Fatalf("unexpected types value: %q", q.Get("types"))
		}
		if q.Get("team_id") != "T123" {
			t.Fatalf("unexpected team_id: %q", q.Get("team_id"))
		}

		requestCount++
		w.Header().Set("Content-Type", "application/json")
		if requestCount == 1 {
			if q.Get("cursor") != "" {
				t.Fatalf("expected first request without cursor, got %q", q.Get("cursor"))
			}
			_, _ = w.Write([]byte(`{"ok":true,"channels":[{"id":"C1"},{"id":"C2"}],"response_metadata":{"next_cursor":"cursor-1"}}`))
			return
		}

		if q.Get("cursor") != "cursor-1" {
			t.Fatalf("expected second request cursor cursor-1, got %q", q.Get("cursor"))
		}
		_, _ = w.Write([]byte(`{"ok":true,"channels":[{"id":"C3"}],"response_metadata":{"next_cursor":""}}`))
	}))
	defer srv.Close()

	client, err := NewClient(
		WithBaseURL(srv.URL),
		WithToken("xoxb-test"),
		WithTeamID("T123"),
		WithTransport(transport.New()),
	)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	channels, err := client.Conversations().GetConversationList(context.Background(), true, []string{"public_channel", "private_channel"})
	if err != nil {
		t.Fatalf("GetConversationList failed: %v", err)
	}
	if len(channels) != 3 {
		t.Fatalf("expected 3 channels, got %d", len(channels))
	}
	if got := channels[2].ID; got != "C3" {
		t.Fatalf("unexpected last channel ID: %s", got)
	}
}

func TestCreateUserGroup(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/usergroups.create" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		values, err := url.ParseQuery(string(bodyBytes))
		if err != nil {
			t.Fatalf("parse query: %v", err)
		}
		if values.Get("name") != "Backend" {
			t.Fatalf("unexpected name: %q", values.Get("name"))
		}
		if values.Get("handle") != "backend" {
			t.Fatalf("unexpected handle: %q", values.Get("handle"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"usergroup":{"id":"S1","name":"Backend","handle":"backend"}}`))
	}))
	defer srv.Close()

	client, err := NewClient(
		WithBaseURL(srv.URL),
		WithToken("xoxb-test"),
		WithTransport(transport.New()),
	)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	group, err := client.UserGroups().CreateUserGroup(context.Background(), "Backend", "backend")
	if err != nil {
		t.Fatalf("CreateUserGroup failed: %v", err)
	}
	if strings.TrimSpace(group.ID) != "S1" {
		t.Fatalf("unexpected group id: %q", group.ID)
	}
}

func TestGetHistory(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/conversations.history" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer xoxb-test" {
			t.Fatalf("unexpected authorization header: %q", got)
		}

		q := r.URL.Query()
		if q.Get("channel") != "C123" {
			t.Fatalf("unexpected channel: %q", q.Get("channel"))
		}
		if q.Get("limit") != "10" {
			t.Fatalf("unexpected limit: %q", q.Get("limit"))
		}
		if q.Get("oldest") != "1234567890.000000" {
			t.Fatalf("unexpected oldest: %q", q.Get("oldest"))
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"messages":[{"type":"message","user":"U1","text":"hello","ts":"1234567890.000001"},{"type":"message","user":"U2","text":"world","ts":"1234567890.000002","thread_ts":"1234567890.000001","reply_count":3}],"has_more":false,"pin_count":1}`))
	}))
	defer srv.Close()

	client, err := NewClient(
		WithBaseURL(srv.URL),
		WithToken("xoxb-test"),
		WithTransport(transport.New()),
	)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	resp, err := client.Conversations().GetHistory(context.Background(), &GetHistoryRequest{
		Channel: "C123",
		Limit:   10,
		Oldest:  "1234567890.000000",
	})
	if err != nil {
		t.Fatalf("GetHistory failed: %v", err)
	}
	if len(resp.Messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(resp.Messages))
	}
	if resp.Messages[0].Text != "hello" {
		t.Fatalf("unexpected first message text: %q", resp.Messages[0].Text)
	}
	if resp.Messages[1].ThreadTS != "1234567890.000001" {
		t.Fatalf("unexpected thread_ts: %q", resp.Messages[1].ThreadTS)
	}
	if resp.Messages[1].ReplyCount != 3 {
		t.Fatalf("unexpected reply_count: %d", resp.Messages[1].ReplyCount)
	}
	if resp.HasMore {
		t.Fatalf("expected has_more=false")
	}
	if resp.PinCount != 1 {
		t.Fatalf("expected pin_count=1, got %d", resp.PinCount)
	}
}

func TestGetHistoryValidation(t *testing.T) {
	t.Parallel()

	client, err := NewClient(WithToken("xoxb-test"), WithTransport(transport.New()))
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	if _, err := client.Conversations().GetHistory(context.Background(), nil); err == nil {
		t.Fatalf("expected error for nil request")
	}
	if _, err := client.Conversations().GetHistory(context.Background(), &GetHistoryRequest{}); err == nil {
		t.Fatalf("expected error for empty channel")
	}
}

func TestGetReplies(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/conversations.replies" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Fatalf("unexpected method: %s", r.Method)
		}

		q := r.URL.Query()
		if q.Get("channel") != "C123" {
			t.Fatalf("unexpected channel: %q", q.Get("channel"))
		}
		if q.Get("ts") != "1234567890.000001" {
			t.Fatalf("unexpected ts: %q", q.Get("ts"))
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"messages":[{"type":"message","user":"U1","text":"parent","ts":"1234567890.000001","thread_ts":"1234567890.000001"},{"type":"message","user":"U2","text":"reply 1","ts":"1234567890.000002","thread_ts":"1234567890.000001"}],"has_more":false}`))
	}))
	defer srv.Close()

	client, err := NewClient(
		WithBaseURL(srv.URL),
		WithToken("xoxb-test"),
		WithTransport(transport.New()),
	)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	resp, err := client.Conversations().GetReplies(context.Background(), &GetRepliesRequest{
		Channel: "C123",
		TS:      "1234567890.000001",
	})
	if err != nil {
		t.Fatalf("GetReplies failed: %v", err)
	}
	if len(resp.Messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(resp.Messages))
	}
	if resp.Messages[0].Text != "parent" {
		t.Fatalf("unexpected parent text: %q", resp.Messages[0].Text)
	}
	if resp.Messages[1].Text != "reply 1" {
		t.Fatalf("unexpected reply text: %q", resp.Messages[1].Text)
	}
}

func TestGetRepliesValidation(t *testing.T) {
	t.Parallel()

	client, err := NewClient(WithToken("xoxb-test"), WithTransport(transport.New()))
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	if _, err := client.Conversations().GetReplies(context.Background(), nil); err == nil {
		t.Fatalf("expected error for nil request")
	}
	if _, err := client.Conversations().GetReplies(context.Background(), &GetRepliesRequest{Channel: "C123"}); err == nil {
		t.Fatalf("expected error for empty ts")
	}
	if _, err := client.Conversations().GetReplies(context.Background(), &GetRepliesRequest{TS: "123.456"}); err == nil {
		t.Fatalf("expected error for empty channel")
	}
}

func TestGetHistorySlackError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":false,"error":"channel_not_found"}`))
	}))
	defer srv.Close()

	client, err := NewClient(
		WithBaseURL(srv.URL),
		WithToken("xoxb-test"),
		WithTransport(transport.New()),
	)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	_, err = client.Conversations().GetHistory(context.Background(), &GetHistoryRequest{Channel: "INVALID"})
	if err == nil {
		t.Fatalf("expected error")
	}
	slackErr, ok := err.(*Error)
	if !ok {
		t.Fatalf("expected slack.Error, got %T", err)
	}
	if slackErr.Code != "channel_not_found" {
		t.Fatalf("unexpected error code: %q", slackErr.Code)
	}
}

func TestSlackClientRespectsBasePath(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat.postMessage" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"channel":"C123","ts":"1.0"}`))
	}))
	defer srv.Close()

	client, err := NewClient(
		WithBaseURL(srv.URL+"/api"),
		WithToken("xoxb-test"),
		WithTransport(transport.New()),
	)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	_, err = client.Messages().PostMessage(context.Background(), &PostMessageRequest{Channel: "C123", Text: "hello"})
	if err != nil {
		t.Fatalf("PostMessage failed: %v", err)
	}
}
