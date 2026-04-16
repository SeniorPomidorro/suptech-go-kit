package slack

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SeniorPomidorro/suptech-go-kit/pkg/transport"
)

func TestListUserGroupUsers(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/usergroups.users.list" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Fatalf("unexpected method: %s", r.Method)
		}

		q := r.URL.Query()
		if q.Get("usergroup") != "S123" {
			t.Fatalf("unexpected usergroup: %q", q.Get("usergroup"))
		}
		if q.Get("include_disabled") != "true" {
			t.Fatalf("unexpected include_disabled: %q", q.Get("include_disabled"))
		}
		if q.Get("team_id") != "T-explicit" {
			t.Fatalf("unexpected team_id: %q", q.Get("team_id"))
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"users":["U1","U2"]}`))
	}))
	defer srv.Close()

	client, err := NewClient(
		WithBaseURL(srv.URL),
		WithToken("xoxb-test"),
		WithTeamID("T-from-client"),
		WithTransport(transport.New()),
	)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	users, err := client.UserGroups().ListUserGroupUsers(context.Background(), &ListUserGroupUsersRequest{
		UserGroup:       "S123",
		IncludeDisabled: true,
		TeamID:          "T-explicit",
	})
	if err != nil {
		t.Fatalf("ListUserGroupUsers failed: %v", err)
	}
	if len(users) != 2 || users[0] != "U1" || users[1] != "U2" {
		t.Fatalf("unexpected users response: %+v", users)
	}
}

func TestListUserGroupUsersValidation(t *testing.T) {
	t.Parallel()

	client, err := NewClient(WithToken("xoxb-test"), WithTransport(transport.New()))
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	if _, err := client.UserGroups().ListUserGroupUsers(context.Background(), nil); err == nil {
		t.Fatalf("expected error for nil request")
	}
	if _, err := client.UserGroups().ListUserGroupUsers(context.Background(), &ListUserGroupUsersRequest{}); err == nil {
		t.Fatalf("expected error for empty usergroup")
	}
}

func TestUpdateUserGroupUsers(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/usergroups.users.update" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}

		if r.Form.Get("usergroup") != "S123" {
			t.Fatalf("unexpected usergroup: %q", r.Form.Get("usergroup"))
		}
		if r.Form.Get("users") != "U1,U2" {
			t.Fatalf("unexpected users: %q", r.Form.Get("users"))
		}
		if r.Form.Get("include_count") != "true" {
			t.Fatalf("unexpected include_count: %q", r.Form.Get("include_count"))
		}
		if r.Form.Get("additional_channels") != "C1,C2" {
			t.Fatalf("unexpected additional_channels: %q", r.Form.Get("additional_channels"))
		}
		if r.Form.Get("is_shared") != "true" {
			t.Fatalf("unexpected is_shared: %q", r.Form.Get("is_shared"))
		}
		if r.Form.Get("team_id") != "T-from-client" {
			t.Fatalf("unexpected team_id: %q", r.Form.Get("team_id"))
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"usergroup":{"id":"S123","name":"Ops","handle":"ops"}}`))
	}))
	defer srv.Close()

	client, err := NewClient(
		WithBaseURL(srv.URL),
		WithToken("xoxb-test"),
		WithTeamID("T-from-client"),
		WithTransport(transport.New()),
	)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	group, err := client.UserGroups().UpdateUserGroupUsers(context.Background(), &UpdateUserGroupUsersRequest{
		UserGroup:          "S123",
		Users:              []string{"U1", "U2"},
		IncludeCount:       true,
		AdditionalChannels: []string{"C1", "C2"},
		IsShared:           true,
	})
	if err != nil {
		t.Fatalf("UpdateUserGroupUsers failed: %v", err)
	}
	if group.ID != "S123" || group.Handle != "ops" {
		t.Fatalf("unexpected usergroup response: %+v", group)
	}
}

func TestUpdateUserGroupUsersValidation(t *testing.T) {
	t.Parallel()

	client, err := NewClient(WithToken("xoxb-test"), WithTransport(transport.New()))
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	if _, err := client.UserGroups().UpdateUserGroupUsers(context.Background(), nil); err == nil {
		t.Fatalf("expected error for nil request")
	}
	if _, err := client.UserGroups().UpdateUserGroupUsers(context.Background(), &UpdateUserGroupUsersRequest{}); err == nil {
		t.Fatalf("expected error for empty usergroup")
	}
	if _, err := client.UserGroups().UpdateUserGroupUsers(context.Background(), &UpdateUserGroupUsersRequest{UserGroup: "S123"}); err == nil {
		t.Fatalf("expected error for empty users list")
	}
}
