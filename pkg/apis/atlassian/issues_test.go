package atlassian

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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

func TestUpdateIssueFields(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/3/issue/PROJ-1" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPut {
			t.Fatalf("unexpected method: %s", r.Method)
		}

		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		fields, ok := payload["fields"].(map[string]any)
		if !ok {
			t.Fatalf("fields block missing")
		}
		if fields["summary"] != "Updated title" {
			t.Fatalf("unexpected summary: %v", fields["summary"])
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	client, err := NewClient(WithBaseURL(srv.URL), WithTransport(transport.New()))
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	_, err = client.Issues().UpdateIssue(context.Background(), "PROJ-1", &UpdateIssueRequest{
		Fields: map[string]any{
			"summary": "Updated title",
		},
	}, nil)
	if err != nil {
		t.Fatalf("UpdateIssue: %v", err)
	}
}

func TestUpdateIssueWithReturnIssue(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("returnIssue") != "true" {
			t.Fatalf("expected returnIssue=true, got %q", r.URL.Query().Get("returnIssue"))
		}
		if r.URL.Query().Get("notifyUsers") != "false" {
			t.Fatalf("expected notifyUsers=false, got %q", r.URL.Query().Get("notifyUsers"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"10001","key":"PROJ-1","fields":{"summary":"Updated"}}`))
	}))
	defer srv.Close()

	client, err := NewClient(WithBaseURL(srv.URL), WithTransport(transport.New()))
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	notify := false
	issue, err := client.Issues().UpdateIssue(context.Background(), "PROJ-1", &UpdateIssueRequest{
		Fields: map[string]any{"summary": "Updated"},
	}, &UpdateIssueOptions{
		NotifyUsers: &notify,
		ReturnIssue: true,
	})
	if err != nil {
		t.Fatalf("UpdateIssue: %v", err)
	}
	if issue == nil {
		t.Fatalf("expected issue to be returned")
	}
	if issue.Key != "PROJ-1" {
		t.Fatalf("unexpected key: %q", issue.Key)
	}
}

func TestUpdateIssueWithADF(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			Fields struct {
				Description json.RawMessage `json:"description"`
			} `json:"fields"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}

		var doc struct {
			Type    string `json:"type"`
			Version int    `json:"version"`
			Content []struct {
				Type    string `json:"type"`
				Content []struct {
					Type string `json:"type"`
					Text string `json:"text"`
				} `json:"content"`
			} `json:"content"`
		}
		if err := json.Unmarshal(payload.Fields.Description, &doc); err != nil {
			t.Fatalf("unmarshal ADF: %v", err)
		}
		if doc.Type != "doc" || doc.Version != 1 {
			t.Fatalf("unexpected ADF doc: type=%q version=%d", doc.Type, doc.Version)
		}
		if len(doc.Content) != 2 {
			t.Fatalf("expected 2 paragraphs, got %d", len(doc.Content))
		}
		if doc.Content[0].Content[0].Text != "First paragraph" {
			t.Fatalf("unexpected first paragraph: %q", doc.Content[0].Content[0].Text)
		}
		if doc.Content[1].Content[0].Text != "Second paragraph" {
			t.Fatalf("unexpected second paragraph: %q", doc.Content[1].Content[0].Text)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	client, err := NewClient(WithBaseURL(srv.URL), WithTransport(transport.New()))
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	_, err = client.Issues().UpdateIssue(context.Background(), "PROJ-1", &UpdateIssueRequest{
		Fields: map[string]any{
			"description": TextToADF("First paragraph\nSecond paragraph"),
		},
	}, nil)
	if err != nil {
		t.Fatalf("UpdateIssue with ADF: %v", err)
	}
}

func TestUpdateIssueUpdateOperations(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		update, ok := payload["update"].(map[string]any)
		if !ok {
			t.Fatalf("update block missing")
		}
		labels, ok := update["labels"].([]any)
		if !ok {
			t.Fatalf("labels ops missing")
		}
		if len(labels) != 1 {
			t.Fatalf("expected 1 label op, got %d", len(labels))
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	client, err := NewClient(WithBaseURL(srv.URL), WithTransport(transport.New()))
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	_, err = client.Issues().UpdateIssue(context.Background(), "PROJ-1", &UpdateIssueRequest{
		Update: map[string][]any{
			"labels": {map[string]string{"add": "critical"}},
		},
	}, nil)
	if err != nil {
		t.Fatalf("UpdateIssue with operations: %v", err)
	}
}

func TestUpdateIssueValidation(t *testing.T) {
	t.Parallel()

	client, err := NewClient(WithBaseURL("https://example.com"), WithTransport(transport.New()))
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	if _, err := client.Issues().UpdateIssue(context.Background(), "", nil, nil); err == nil {
		t.Fatalf("expected error for empty ticket key")
	}
	if _, err := client.Issues().UpdateIssue(context.Background(), "PROJ-1", nil, nil); err == nil {
		t.Fatalf("expected error for nil body")
	}
}

func TestTextToADF(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		input      string
		wantParas  int
		wantTexts  []string
	}{
		{"single line", "hello", 1, []string{"hello"}},
		{"two lines", "first\nsecond", 2, []string{"first", "second"}},
		{"empty line spacing", "before\n\nafter", 3, []string{"before", "", "after"}},
		{"empty string", "", 1, []string{""}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			raw := TextToADF(tt.input)

			var doc struct {
				Type    string `json:"type"`
				Version int    `json:"version"`
				Content []struct {
					Type    string `json:"type"`
					Content []struct {
						Text string `json:"text"`
					} `json:"content"`
				} `json:"content"`
			}
			if err := json.Unmarshal(raw, &doc); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			if doc.Type != "doc" || doc.Version != 1 {
				t.Fatalf("unexpected doc: type=%q version=%d", doc.Type, doc.Version)
			}
			if len(doc.Content) != tt.wantParas {
				t.Fatalf("expected %d paragraphs, got %d", tt.wantParas, len(doc.Content))
			}
			for i, want := range tt.wantTexts {
				var got string
				if len(doc.Content[i].Content) > 0 {
					got = doc.Content[i].Content[0].Text
				}
				if got != want {
					t.Fatalf("paragraph %d: expected %q, got %q", i, want, got)
				}
			}
		})
	}
}

func TestADFToText(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		adf  string
		want string
	}{
		{
			"simple paragraphs",
			`{"type":"doc","version":1,"content":[{"type":"paragraph","content":[{"type":"text","text":"hello"}]},{"type":"paragraph","content":[{"type":"text","text":"world"}]}]}`,
			"hello\nworld",
		},
		{
			"mixed inline marks",
			`{"type":"doc","version":1,"content":[{"type":"paragraph","content":[{"type":"text","text":"normal "},{"type":"text","text":"bold","marks":[{"type":"strong"}]},{"type":"text","text":" end"}]}]}`,
			"normal bold end",
		},
		{
			"empty doc",
			`{"type":"doc","version":1,"content":[]}`,
			"",
		},
		{
			"invalid json",
			`not json`,
			"",
		},
		{
			"empty input",
			"",
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ADFToText(json.RawMessage(tt.adf))
			if got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestTextToADFRoundTrip(t *testing.T) {
	t.Parallel()

	input := "First line\nSecond line\nThird line"
	adf := TextToADF(input)
	got := ADFToText(adf)

	if got != input {
		t.Fatalf("round-trip failed:\ninput:  %q\noutput: %q", input, got)
	}
}

func TestTextToADFRoundTripWithEmptyLines(t *testing.T) {
	t.Parallel()

	input := "Before\n\nAfter"
	adf := TextToADF(input)
	got := ADFToText(adf)

	// Empty paragraphs produce empty lines in the round-trip.
	if !strings.Contains(got, "Before") || !strings.Contains(got, "After") {
		t.Fatalf("round-trip lost content:\ninput:  %q\noutput: %q", input, got)
	}
}
