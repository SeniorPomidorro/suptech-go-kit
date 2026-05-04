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

func TestCreateIssue(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/api/3/issue" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
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
		if fields["summary"] != "New task" {
			t.Fatalf("unexpected summary: %v", fields["summary"])
		}
		project, ok := fields["project"].(map[string]any)
		if !ok {
			t.Fatalf("project block missing")
		}
		if project["key"] != "PROJ" {
			t.Fatalf("unexpected project key: %v", project["key"])
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"10001","key":"PROJ-1","self":"https://example.atlassian.net/rest/api/3/issue/10001"}`))
	}))
	defer srv.Close()

	client, err := NewClient(WithBaseURL(srv.URL), WithTransport(transport.New()))
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	created, err := client.Issues().CreateIssue(context.Background(), &CreateIssueRequest{
		Fields: map[string]any{
			"summary":   "New task",
			"project":   map[string]any{"key": "PROJ"},
			"issuetype": map[string]any{"name": "Task"},
		},
	})
	if err != nil {
		t.Fatalf("CreateIssue: %v", err)
	}
	if created.Key != "PROJ-1" {
		t.Fatalf("unexpected key: %q", created.Key)
	}
	if created.ID != "10001" {
		t.Fatalf("unexpected id: %q", created.ID)
	}
}

func TestCreateIssueWithADF(t *testing.T) {
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
		}
		if err := json.Unmarshal(payload.Fields.Description, &doc); err != nil {
			t.Fatalf("unmarshal ADF: %v", err)
		}
		if doc.Type != "doc" || doc.Version != 1 {
			t.Fatalf("expected ADF doc, got type=%q version=%d", doc.Type, doc.Version)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"id":"10002","key":"PROJ-2"}`))
	}))
	defer srv.Close()

	client, err := NewClient(WithBaseURL(srv.URL), WithTransport(transport.New()))
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	created, err := client.Issues().CreateIssue(context.Background(), &CreateIssueRequest{
		Fields: map[string]any{
			"summary":     "Task with description",
			"project":     map[string]any{"key": "PROJ"},
			"issuetype":   map[string]any{"name": "Task"},
			"description": TextToADF("Task description\nSecond line"),
		},
	})
	if err != nil {
		t.Fatalf("CreateIssue with ADF: %v", err)
	}
	if created.Key != "PROJ-2" {
		t.Fatalf("unexpected key: %q", created.Key)
	}
}

func TestCreateIssueValidation(t *testing.T) {
	t.Parallel()

	client, err := NewClient(WithBaseURL("https://example.com"), WithTransport(transport.New()))
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	if _, err := client.Issues().CreateIssue(context.Background(), nil); err == nil {
		t.Fatalf("expected error for nil body")
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

func TestGetTransitions(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/rest/api/3/issue/ABC-1/transitions" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("includeUnavailableTransitions"); got != "true" {
			t.Fatalf("unexpected includeUnavailableTransitions: %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"transitions":[{"id":"11","name":"Reopen","to":{"id":"5","name":"Reopened"},"isAvailable":true}]}`))
	}))
	defer srv.Close()

	client, err := NewClient(WithBaseURL(srv.URL), WithTransport(transport.New()))
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	list, err := client.Issues().GetTransitions(context.Background(), "ABC-1", &GetTransitionsOptions{
		IncludeUnavailableTransitions: true,
	})
	if err != nil {
		t.Fatalf("GetTransitions: %v", err)
	}
	if len(list.Transitions) != 1 || list.Transitions[0].ID != "11" || list.Transitions[0].To.Name != "Reopened" {
		t.Fatalf("unexpected transitions: %+v", list.Transitions)
	}
	// Without Expand="transitions.fields" the upstream omits per-field
	// metadata; the parser must keep Fields nil so callers can rely on
	// `len(Fields)==0` to gate UI rendering of transition screens.
	if got := list.Transitions[0].Fields; got != nil {
		t.Fatalf("expected Fields to be nil without expand, got: %+v", got)
	}
}

func TestGetTransitionsExpandsFields(t *testing.T) {
	t.Parallel()

	const respBody = `{
		"expand": "transitions",
		"transitions": [
			{
				"id": "91",
				"name": "Identified as an Incident",
				"to": {"id": "10482", "name": "Active"},
				"hasScreen": true,
				"isAvailable": true,
				"fields": {
					"summary": {
						"required": true,
						"name": "Summary",
						"key": "summary",
						"operations": ["set"],
						"schema": {"type": "string", "system": "summary"},
						"hasDefaultValue": false
					},
					"priority": {
						"required": true,
						"name": "Priority",
						"key": "priority",
						"operations": ["set"],
						"schema": {"type": "priority", "system": "priority"},
						"allowedValues": [
							{"id": "1", "name": "Highest"},
							{"id": "3", "name": "Medium"}
						]
					},
					"assignee": {
						"required": false,
						"name": "Assignee",
						"key": "assignee",
						"operations": ["set"],
						"schema": {"type": "user", "system": "assignee"},
						"autoCompleteUrl": "https://x.atlassian.net/rest/api/2/user/assignable/search?issueKey=ABC-1&username="
					},
					"customfield_10090": {
						"required": true,
						"name": "Affected services",
						"key": "customfield_10090",
						"operations": ["add", "set", "remove"],
						"schema": {
							"type": "array",
							"items": "option",
							"custom": "com.atlassian.jira.plugin.system.customfieldtypes:multiselect",
							"customId": 10090
						},
						"allowedValues": [
							{"id": "10100", "value": "billing"},
							{"id": "10101", "value": "checkout"}
						]
					}
				}
			}
		]
	}`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("expand"); got != "transitions.fields" {
			t.Fatalf("unexpected expand: %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(respBody))
	}))
	defer srv.Close()

	client, err := NewClient(WithBaseURL(srv.URL), WithTransport(transport.New()))
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	list, err := client.Issues().GetTransitions(context.Background(), "ABC-1", &GetTransitionsOptions{
		Expand: "transitions.fields",
	})
	if err != nil {
		t.Fatalf("GetTransitions: %v", err)
	}
	if len(list.Transitions) != 1 {
		t.Fatalf("expected 1 transition, got %d", len(list.Transitions))
	}
	tr := list.Transitions[0]
	if !tr.HasScreen {
		t.Fatalf("expected HasScreen=true")
	}
	if len(tr.Fields) != 4 {
		t.Fatalf("expected 4 fields, got %d: %+v", len(tr.Fields), tr.Fields)
	}

	summary, ok := tr.Fields["summary"]
	if !ok {
		t.Fatalf("missing summary field")
	}
	if !summary.Required || summary.Schema.Type != "string" || summary.Schema.System != "summary" {
		t.Fatalf("unexpected summary field: %+v", summary)
	}

	priority := tr.Fields["priority"]
	if priority.Schema.Type != "priority" || len(priority.AllowedValues) != 2 {
		t.Fatalf("unexpected priority field: %+v", priority)
	}
	if priority.AllowedValues[0].ID != "1" || priority.AllowedValues[0].Name != "Highest" {
		t.Fatalf("unexpected priority allowedValues: %+v", priority.AllowedValues)
	}

	assignee := tr.Fields["assignee"]
	if assignee.Schema.Type != "user" || assignee.Required {
		t.Fatalf("unexpected assignee field: %+v", assignee)
	}
	if assignee.AutoCompleteURL == "" {
		t.Fatalf("expected autoCompleteUrl to be parsed for assignee")
	}

	multi := tr.Fields["customfield_10090"]
	if multi.Schema.Type != "array" || multi.Schema.Items != "option" {
		t.Fatalf("unexpected multiselect schema: %+v", multi.Schema)
	}
	if multi.Schema.CustomID != 10090 {
		t.Fatalf("expected customId=10090, got %d", multi.Schema.CustomID)
	}
	if len(multi.AllowedValues) != 2 || multi.AllowedValues[0].Value != "billing" {
		t.Fatalf("unexpected option allowedValues: %+v", multi.AllowedValues)
	}
}

func TestDoTransition(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/rest/api/3/issue/ABC-1/transitions" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		transition, ok := payload["transition"].(map[string]any)
		if !ok {
			t.Fatalf("transition block is required")
		}
		if id, _ := transition["id"].(string); id != "31" {
			t.Fatalf("unexpected transition id: %q", id)
		}
		if _, ok := payload["fields"].(map[string]any); !ok {
			t.Fatalf("fields block is required")
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	client, err := NewClient(WithBaseURL(srv.URL), WithTransport(transport.New()))
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	err = client.Issues().DoTransition(context.Background(), "ABC-1", &DoTransitionRequest{
		TransitionID: "31",
		Fields:       map[string]any{"resolution": map[string]string{"name": "Done"}},
	})
	if err != nil {
		t.Fatalf("DoTransition: %v", err)
	}
}

func TestTransitionsValidation(t *testing.T) {
	t.Parallel()

	client, err := NewClient(WithBaseURL("https://example.com"), WithTransport(transport.New()))
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	if _, err := client.Issues().GetTransitions(context.Background(), "", nil); err == nil {
		t.Fatalf("expected error for empty ticket key")
	}
	if err := client.Issues().DoTransition(context.Background(), "", &DoTransitionRequest{TransitionID: "1"}); err == nil {
		t.Fatalf("expected error for empty ticket key")
	}
	if err := client.Issues().DoTransition(context.Background(), "PROJ-1", nil); err == nil {
		t.Fatalf("expected error for nil body")
	}
	if err := client.Issues().DoTransition(context.Background(), "PROJ-1", &DoTransitionRequest{}); err == nil {
		t.Fatalf("expected error for empty transition id")
	}
}

