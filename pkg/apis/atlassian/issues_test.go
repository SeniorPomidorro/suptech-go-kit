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

	requestCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if r.URL.Path != "/rest/api/3/search/jql" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		var payload struct {
			NextPageToken string   `json:"nextPageToken"`
			MaxResults    int      `json:"maxResults"`
			Fields        []string `json:"fields"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if payload.MaxResults != 2 {
			t.Fatalf("unexpected maxResults: %d", payload.MaxResults)
		}
		if len(payload.Fields) != 1 || payload.Fields[0] != "*all" {
			t.Fatalf("unexpected fields: %v", payload.Fields)
		}

		w.Header().Set("Content-Type", "application/json")
		if payload.NextPageToken == "" {
			_, _ = w.Write([]byte(`{"nextPageToken":"page2token","issues":[{"id":"1","key":"ABC-1"},{"id":"2","key":"ABC-2"}]}`))
			return
		}
		if payload.NextPageToken != "page2token" {
			t.Fatalf("unexpected nextPageToken: %q", payload.NextPageToken)
		}
		_, _ = w.Write([]byte(`{"isLast":true,"issues":[{"id":"3","key":"ABC-3"}]}`))
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
	if requestCount != 2 {
		t.Fatalf("expected 2 paginated requests, got %d", requestCount)
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
