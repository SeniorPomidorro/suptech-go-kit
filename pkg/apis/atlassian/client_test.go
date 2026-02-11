package atlassian

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SeniorPomidorro/suptech-go-kit/pkg/transport"
)

func TestClientAuthHeaders(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		auth       Auth
		expectAuth string
	}{
		{
			name:       "basic email token",
			auth:       Auth{Mode: AuthBasicEmailToken, Email: "user@example.com", Token: "secret"},
			expectAuth: "Basic " + base64.StdEncoding.EncodeToString([]byte("user@example.com:secret")),
		},
		{
			name:       "bearer",
			auth:       Auth{Mode: AuthBearerToken, Token: "token-1"},
			expectAuth: "Bearer token-1",
		},
		{
			name:       "basic direct token",
			auth:       Auth{Mode: AuthBasicToken, Token: "encoded-token"},
			expectAuth: "Basic encoded-token",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if got := r.Header.Get("Authorization"); got != tc.expectAuth {
					t.Fatalf("unexpected auth header: %q", got)
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"id":"1","key":"ABC-1"}`))
			}))
			defer srv.Close()

			client, err := NewClient(
				WithBaseURL(srv.URL),
				WithAuth(tc.auth),
				WithTransport(transport.New()),
			)
			if err != nil {
				t.Fatalf("new client: %v", err)
			}

			if _, err := client.Issues().GetIssue(context.Background(), "ABC-1"); err != nil {
				t.Fatalf("GetIssue failed: %v", err)
			}
		})
	}
}

func TestNewClientValidatesBaseURL(t *testing.T) {
	t.Parallel()

	_, err := NewClient(WithBaseURL("invalid-url"))
	if err == nil {
		t.Fatalf("expected error")
	}
	if got := err.Error(); got == "" {
		t.Fatalf("expected non-empty error")
	}
}

func TestClientBuildsRequestURL(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/3/issue/ABC-2" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"2","key":"ABC-2"}`))
	}))
	defer srv.Close()

	client, err := NewClient(WithBaseURL(srv.URL), WithTransport(transport.New()))
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	issue, err := client.Issues().GetIssue(context.Background(), "ABC-2")
	if err != nil {
		t.Fatalf("GetIssue: %v", err)
	}
	if issue.Key != "ABC-2" {
		t.Fatalf("unexpected issue key: %s", issue.Key)
	}
}

func TestDoNoResponseBody(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	client, err := NewClient(WithBaseURL(srv.URL), WithTransport(transport.New()))
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPut, fmt.Sprintf("%s/test", srv.URL), nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	if err := client.doNoResponseBody(req); err != nil {
		t.Fatalf("doNoResponseBody: %v", err)
	}
}
