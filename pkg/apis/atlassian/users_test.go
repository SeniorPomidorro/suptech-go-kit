package atlassian

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

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

	users, err := client.Users().FindUsers(context.Background(), "john", FindUsersOptions{MaxResults: 5})
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
