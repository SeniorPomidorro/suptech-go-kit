package atlassian

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/SeniorPomidorro/suptech-go-kit/pkg/transport"
)

func TestFindUsers(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("query"); got != "john" {
			t.Fatalf("unexpected query: %q", got)
		}
		if got := r.URL.Query().Get("maxResults"); got != "5" {
			t.Fatalf("unexpected maxResults: %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"accountId":"1","displayName":"John"}]`))
	}))
	defer srv.Close()

	client, err := NewClient(WithBaseURL(srv.URL), WithTransport(transport.New()))
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	users, err := client.Users().FindUsers(context.Background(), "john", &FindUsersOptions{MaxResults: 5})
	if err != nil {
		t.Fatalf("FindUsers: %v", err)
	}
	if len(users) != 1 {
		t.Fatalf("expected 1 user, got %d", len(users))
	}
	if users[0].DisplayName != "John" {
		t.Fatalf("unexpected display name: %s", users[0].DisplayName)
	}
}

func TestBulkGetUsers(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/3/user/bulk" {
			t.Fatalf("unexpected path: %q", r.URL.Path)
		}
		if got := r.URL.Query().Get("startAt"); got != "2" {
			t.Fatalf("unexpected startAt: %q", got)
		}
		if got := r.URL.Query().Get("maxResults"); got != "20" {
			t.Fatalf("unexpected maxResults: %q", got)
		}

		accountIDs := r.URL.Query()["accountId"]
		if len(accountIDs) != 2 || accountIDs[0] != "acc-1" || accountIDs[1] != "acc-2" {
			t.Fatalf("unexpected accountId query params: %+v", accountIDs)
		}
		usernames := r.URL.Query()["username"]
		if len(usernames) != 2 || usernames[0] != "john" || usernames[1] != "jane" {
			t.Fatalf("unexpected username query params: %+v", usernames)
		}
		keys := r.URL.Query()["key"]
		if len(keys) != 2 || keys[0] != "legacy-1" || keys[1] != "legacy-2" {
			t.Fatalf("unexpected key query params: %+v", keys)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"self":"https://example.atlassian.net/rest/api/3/user/bulk","nextPage":"https://example.atlassian.net/rest/api/3/user/bulk?startAt=22","maxResults":20,"startAt":2,"total":23,"isLast":false,"values":[{"accountId":"acc-1","displayName":"John Doe","emailAddress":"john@example.com","active":true}]}`))
	}))
	defer srv.Close()

	client, err := NewClient(WithBaseURL(srv.URL), WithTransport(transport.New()))
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	result, err := client.Users().BulkGetUsers(context.Background(), &BulkGetUsersOptions{
		StartAt:    2,
		MaxResults: 20,
		AccountIDs: []string{"acc-1", "acc-2"},
		Usernames:  []string{"john", "jane"},
		Keys:       []string{"legacy-1", "legacy-2"},
	})
	if err != nil {
		t.Fatalf("BulkGetUsers: %v", err)
	}
	if result.StartAt != 2 || result.MaxResults != 20 || result.Total != 23 {
		t.Fatalf("unexpected pagination data: %+v", result)
	}
	if result.IsLast {
		t.Fatalf("expected isLast=false")
	}
	if len(result.Values) != 1 {
		t.Fatalf("expected 1 user, got %d", len(result.Values))
	}
	if result.Values[0].AccountID != "acc-1" {
		t.Fatalf("unexpected account ID: %q", result.Values[0].AccountID)
	}
}

func TestBulkGetUsersValidation(t *testing.T) {
	t.Parallel()

	client, err := NewClient(
		WithBaseURL("https://example.atlassian.net"),
		WithTransport(transport.New()),
	)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	if _, err := client.Users().BulkGetUsers(context.Background(), nil); err == nil {
		t.Fatalf("expected error for nil options")
	}
	if _, err := client.Users().BulkGetUsers(context.Background(), &BulkGetUsersOptions{}); err == nil {
		t.Fatalf("expected error for missing account IDs")
	}
	if _, err := client.Users().BulkGetUsers(context.Background(), &BulkGetUsersOptions{AccountIDs: []string{" ", "\t"}}); err == nil {
		t.Fatalf("expected error for empty account IDs")
	}
}

func TestBulkGetUsersFetchAll(t *testing.T) {
	t.Parallel()

	requests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/3/user/bulk" {
			t.Fatalf("unexpected path: %q", r.URL.Path)
		}
		if got := r.URL.Query().Get("maxResults"); got != "2" {
			t.Fatalf("unexpected maxResults: %q", got)
		}
		accountIDs := r.URL.Query()["accountId"]
		if len(accountIDs) != 2 || accountIDs[0] != "acc-1" || accountIDs[1] != "acc-2" {
			t.Fatalf("unexpected accountId params: %+v", accountIDs)
		}

		requests++
		w.Header().Set("Content-Type", "application/json")
		if requests == 1 {
			if got := r.URL.Query().Get("startAt"); got != "" {
				t.Fatalf("expected first request without startAt, got %q", got)
			}
			_, _ = w.Write([]byte(`{"self":"https://example.atlassian.net/rest/api/3/user/bulk","nextPage":"https://example.atlassian.net/rest/api/3/user/bulk?startAt=2","maxResults":2,"startAt":0,"total":3,"isLast":false,"values":[{"accountId":"acc-1","displayName":"John Doe","active":true},{"accountId":"acc-2","displayName":"Jane Doe","active":true}]}`))
			return
		}

		if got := r.URL.Query().Get("startAt"); got != "2" {
			t.Fatalf("expected second request startAt=2, got %q", got)
		}
		_, _ = w.Write([]byte(`{"self":"https://example.atlassian.net/rest/api/3/user/bulk","maxResults":2,"startAt":2,"total":3,"isLast":true,"values":[{"accountId":"acc-3","displayName":"Bob Doe","active":true}]}`))
	}))
	defer srv.Close()

	client, err := NewClient(WithBaseURL(srv.URL), WithTransport(transport.New()))
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	result, err := client.Users().BulkGetUsers(context.Background(), &BulkGetUsersOptions{
		MaxResults:       2,
		AccountIDs:       []string{"acc-1", "acc-2"},
		FetchAll:         true,
		FetchAllThrottle: time.Millisecond,
	})
	if err != nil {
		t.Fatalf("BulkGetUsers: %v", err)
	}
	if requests != 2 {
		t.Fatalf("expected 2 requests, got %d", requests)
	}
	if len(result.Values) != 3 {
		t.Fatalf("expected 3 users, got %d", len(result.Values))
	}
	if result.Values[2].AccountID != "acc-3" {
		t.Fatalf("unexpected last account ID: %q", result.Values[2].AccountID)
	}
	if !result.IsLast {
		t.Fatalf("expected isLast=true")
	}
	if result.NextPage != "" {
		t.Fatalf("expected nextPage empty, got %q", result.NextPage)
	}
}

func TestListGroupMembers(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/3/group/member" {
			t.Fatalf("unexpected path: %q", r.URL.Path)
		}
		if got := r.URL.Query().Get("groupId"); got != "grp-1" {
			t.Fatalf("unexpected groupId: %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Query().Get("startAt") {
		case "0":
			_, _ = w.Write([]byte(`{"startAt":0,"isLast":false,"values":[{"accountId":"a1","emailAddress":"one@example.com"},{"accountId":"a2"}]}`))
		case "2":
			_, _ = w.Write([]byte(`{"startAt":2,"isLast":true,"values":[{"accountId":"a3"}]}`))
		default:
			t.Fatalf("unexpected startAt: %q", r.URL.Query().Get("startAt"))
		}
	}))
	defer srv.Close()

	client, err := NewClient(WithBaseURL(srv.URL), WithTransport(transport.New()))
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	members, err := client.Users().ListGroupMembers(ctx, " grp-1 ")
	if err != nil {
		t.Fatalf("ListGroupMembers: %v", err)
	}
	if len(members) != 3 || members[0].Email != "one@example.com" || members[2].AccountID != "a3" {
		t.Fatalf("unexpected members: %+v", members)
	}
}

func TestListGroupMembersRequiresGroupID(t *testing.T) {
	t.Parallel()

	client, err := NewClient(WithBaseURL("https://example.atlassian.net"), WithTransport(transport.New()))
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	if _, err := client.Users().ListGroupMembers(context.Background(), "  "); err == nil {
		t.Fatal("expected error for empty group ID")
	}
}

func TestListGroupMembersStopsWithoutIsLast(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		pages map[string]string
	}{
		"empty_page_ends_listing": {pages: map[string]string{
			"0": `{"values":[{"accountId":"a1"},{"accountId":"a2"}]}`,
			"2": `{"values":[]}`,
		}},
		"total_reached_ends_listing": {pages: map[string]string{
			"0": `{"total":2,"values":[{"accountId":"a1"},{"accountId":"a2"}]}`,
		}},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, ok := tt.pages[r.URL.Query().Get("startAt")]
				if !ok {
					t.Errorf("unexpected startAt: %q", r.URL.Query().Get("startAt"))
					http.Error(w, "unexpected page", http.StatusBadRequest)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(body))
			}))
			defer srv.Close()

			client, err := NewClient(WithBaseURL(srv.URL), WithTransport(transport.New()))
			if err != nil {
				t.Fatalf("new client: %v", err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			members, err := client.Users().ListGroupMembers(ctx, "grp-1")
			if err != nil {
				t.Fatalf("ListGroupMembers: %v", err)
			}
			if len(members) != 2 {
				t.Fatalf("expected 2 members, got %d", len(members))
			}
		})
	}
}
