package slack

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SeniorPomidorro/suptech-go-kit/pkg/transport"
)

func TestListUserGroupsAddsTeamID(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/usergroups.list" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		if r.Form.Get("team_id") != "T999" {
			t.Fatalf("expected team_id=T999, got %q", r.Form.Get("team_id"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"usergroups":[{"id":"S1","name":"Ops"}]}`))
	}))
	defer srv.Close()

	client, err := NewClient(
		WithBaseURL(srv.URL),
		WithToken("xoxb-test"),
		WithTeamID("T999"),
		WithTransport(transport.New()),
	)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	groups, err := client.UserGroups().ListUserGroups(context.Background())
	if err != nil {
		t.Fatalf("ListUserGroups failed: %v", err)
	}
	if len(groups) != 1 || groups[0].ID != "S1" {
		t.Fatalf("unexpected groups response: %+v", groups)
	}
}

func TestInviteUsersToChannelEncodesUsersList(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/conversations.invite" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		if r.Form.Get("channel") != "C777" {
			t.Fatalf("unexpected channel: %q", r.Form.Get("channel"))
		}
		if r.Form.Get("users") != "U1,U2,U3" {
			t.Fatalf("unexpected users payload: %q", r.Form.Get("users"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"channel":{"id":"C777","name":"alerts"}}`))
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

	channel, err := client.Conversations().InviteUsersToChannel(context.Background(), []string{"U1", "U2", "U3"}, "C777")
	if err != nil {
		t.Fatalf("InviteUsersToChannel failed: %v", err)
	}
	if channel.ID != "C777" {
		t.Fatalf("unexpected channel id: %q", channel.ID)
	}
}

func TestGetUsersByGroupIDResolvesUsers(t *testing.T) {
	t.Parallel()

	usersInfoCalls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/usergroups.users.list":
			if err := r.ParseForm(); err != nil {
				t.Fatalf("parse form: %v", err)
			}
			if r.Form.Get("usergroup") != "S111" {
				t.Fatalf("unexpected usergroup: %q", r.Form.Get("usergroup"))
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"ok":true,"users":["U1","U2"]}`))
		case "/users.info":
			if err := r.ParseForm(); err != nil {
				t.Fatalf("parse form: %v", err)
			}
			usersInfoCalls++
			switch r.Form.Get("user") {
			case "U1":
				_, _ = w.Write([]byte(`{"ok":true,"user":{"id":"U1","name":"alice"}}`))
			case "U2":
				_, _ = w.Write([]byte(`{"ok":true,"user":{"id":"U2","name":"bob"}}`))
			default:
				t.Fatalf("unexpected users.info user: %q", r.Form.Get("user"))
			}
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
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

	users, err := client.Users().GetUsersByGroupID(context.Background(), "S111")
	if err != nil {
		t.Fatalf("GetUsersByGroupID failed: %v", err)
	}
	if len(users) != 2 {
		t.Fatalf("expected 2 users, got %d", len(users))
	}
	if usersInfoCalls != 2 {
		t.Fatalf("expected 2 users.info calls, got %d", usersInfoCalls)
	}
}

func TestGetUsersByGroupIDEmptyListSkipsUsersInfo(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/usergroups.users.list" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"users":[]}`))
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

	users, err := client.Users().GetUsersByGroupID(context.Background(), "S-empty")
	if err != nil {
		t.Fatalf("GetUsersByGroupID failed: %v", err)
	}
	if len(users) != 0 {
		t.Fatalf("expected 0 users, got %d", len(users))
	}
}

func TestPostMessageUsesJSONPayload(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat.postMessage" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Fatalf("unexpected content-type: %q", ct)
		}
		payloadBytes, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("read body: %v", err)
		}
		var payload map[string]any
		if err := json.Unmarshal(payloadBytes, &payload); err != nil {
			t.Fatalf("decode json: %v", err)
		}
		if payload["channel"] != "C1" || payload["text"] != "hello" {
			t.Fatalf("unexpected payload: %+v", payload)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"channel":"C1","ts":"123.456","message":{"text":"hello"}}`))
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

	result, err := client.Messages().PostMessage(context.Background(), "C1", "hello")
	if err != nil {
		t.Fatalf("PostMessage failed: %v", err)
	}
	if result.TS != "123.456" {
		t.Fatalf("unexpected TS: %q", result.TS)
	}
}

func TestSlackMethodReturnsTransportAPIErrorOnNon2xx(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/conversations.create" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write([]byte("rate limited"))
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

	_, err = client.Conversations().CreateConversation(context.Background(), "ops-alerts", false)
	if err == nil {
		t.Fatalf("expected error")
	}

	var apiErr *transport.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected transport.APIError, got %T", err)
	}
	if apiErr.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("unexpected status code: %d", apiErr.StatusCode)
	}
}
