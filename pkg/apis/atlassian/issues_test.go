package atlassian

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SeniorPomidorro/suptech-go-kit/pkg/transport"
)

func TestFindIssuesFetchAll(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/3/search/jql" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		var payload struct {
			StartAt int `json:"startAt"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		if payload.StartAt == 0 {
			_, _ = w.Write([]byte(`{"startAt":0,"maxResults":2,"total":3,"issues":[{"id":"1","key":"ABC-1"},{"id":"2","key":"ABC-2"}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"startAt":2,"maxResults":2,"total":3,"issues":[{"id":"3","key":"ABC-3"}]}`))
	}))
	defer srv.Close()

	client, err := NewClient(WithBaseURL(srv.URL), WithTransport(transport.New()))
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	result, err := client.Issues().FindIssues(context.Background(), "project = ABC", &FindIssuesOptions{
		PageSize: 2,
		FetchAll: true,
	})
	if err != nil {
		t.Fatalf("FindIssues: %v", err)
	}

	if len(result.Issues) != 3 {
		t.Fatalf("expected 3 issues, got %d", len(result.Issues))
	}
	if result.Total != 3 {
		t.Fatalf("expected total=3, got %d", result.Total)
	}
}

func TestManageTagsReplace(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		fields, ok := payload["fields"].(map[string]any)
		if !ok {
			t.Fatalf("fields block is required")
		}
		labels, ok := fields["labels"].([]any)
		if !ok {
			t.Fatalf("labels block is required")
		}
		if len(labels) != 2 {
			t.Fatalf("unexpected labels len: %d", len(labels))
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	client, err := NewClient(WithBaseURL(srv.URL), WithTransport(transport.New()))
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	err = client.Issues().ManageTags(context.Background(), "ABC-1", nil, nil, []string{"backend", "urgent"})
	if err != nil {
		t.Fatalf("ManageTags: %v", err)
	}
}
