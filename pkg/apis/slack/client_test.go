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

	_, err = client.Messages().PostMessage(context.Background(), "C123", "hello")
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
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer xoxb-test" {
			t.Fatalf("unexpected authorization header: %q", got)
		}

		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		if r.Form.Get("types") != "public_channel,private_channel" {
			t.Fatalf("unexpected types value: %q", r.Form.Get("types"))
		}
		if r.Form.Get("team_id") != "T123" {
			t.Fatalf("unexpected team_id: %q", r.Form.Get("team_id"))
		}

		requestCount++
		w.Header().Set("Content-Type", "application/json")
		if requestCount == 1 {
			if r.Form.Get("cursor") != "" {
				t.Fatalf("expected first request without cursor, got %q", r.Form.Get("cursor"))
			}
			_, _ = w.Write([]byte(`{"ok":true,"channels":[{"id":"C1"},{"id":"C2"}],"response_metadata":{"next_cursor":"cursor-1"}}`))
			return
		}

		if r.Form.Get("cursor") != "cursor-1" {
			t.Fatalf("expected second request cursor cursor-1, got %q", r.Form.Get("cursor"))
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

	_, err = client.Messages().PostMessage(context.Background(), "C123", "hello")
	if err != nil {
		t.Fatalf("PostMessage failed: %v", err)
	}
}
