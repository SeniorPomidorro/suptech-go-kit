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

func TestGetObjectBuildsAssetsURL(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wantPath := "/ex/jira/cloud-1/jsm/assets/workspace/ws-9/v1/object/42"
		if r.URL.Path != wantPath {
			t.Fatalf("unexpected path: got=%s want=%s", r.URL.Path, wantPath)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"42","label":"Server-42"}`))
	}))
	defer srv.Close()

	client, err := NewClient(
		WithBaseURL(srv.URL),
		WithCloudBaseURL(srv.URL),
		WithAssetsCloudID("cloud-1"),
		WithAssetsWorkspaceID("ws-9"),
		WithTransport(transport.New()),
	)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	object, err := client.Assets().GetObject(context.Background(), "42")
	if err != nil {
		t.Fatalf("GetObject failed: %v", err)
	}
	if object.ID != "42" {
		t.Fatalf("unexpected object id: %q", object.ID)
	}
}

func TestSearchObjectsAQLFetchAll(t *testing.T) {
	t.Parallel()

	requestCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if r.URL.Path != "/ex/jira/cloud-7/jsm/assets/workspace/ws-7/v1/object/aql" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		var payload struct {
			StartAt int    `json:"startAt"`
			Query   string `json:"qlQuery"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}
		if payload.Query != "Name LIKE server*" {
			t.Fatalf("unexpected query: %q", payload.Query)
		}

		w.Header().Set("Content-Type", "application/json")
		if payload.StartAt == 0 {
			_, _ = w.Write([]byte(`{"startAt":0,"maxResults":2,"total":3,"isLast":false,"values":[{"id":"1"},{"id":"2"}]}`))
			return
		}
		_, _ = w.Write([]byte(`{"startAt":2,"maxResults":2,"total":3,"isLast":true,"values":[{"id":"3"}]}`))
	}))
	defer srv.Close()

	client, err := NewClient(
		WithBaseURL(srv.URL),
		WithCloudBaseURL(srv.URL),
		WithAssetsCloudID("cloud-7"),
		WithAssetsWorkspaceID("ws-7"),
		WithTransport(transport.New()),
	)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	result, err := client.Assets().SearchObjectsAQL(context.Background(), "Name LIKE server*", AssetsSearchOptions{
		PageSize: 2,
		FetchAll: true,
	})
	if err != nil {
		t.Fatalf("SearchObjectsAQL failed: %v", err)
	}
	if len(result.Values) != 3 {
		t.Fatalf("expected 3 objects, got %d", len(result.Values))
	}
	if requestCount != 2 {
		t.Fatalf("expected 2 paginated requests, got %d", requestCount)
	}
}

func TestAssetsPathRequiresCloudAndWorkspace(t *testing.T) {
	t.Parallel()

	clientCloudMissing, err := NewClient(
		WithBaseURL("https://example.atlassian.net"),
		WithAssetsWorkspaceID("ws-1"),
		WithTransport(transport.New()),
	)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	_, err = clientCloudMissing.Assets().GetObject(context.Background(), "1")
	if err == nil || !strings.Contains(err.Error(), "assets cloud ID is required") {
		t.Fatalf("expected cloud ID error, got: %v", err)
	}

	clientWorkspaceMissing, err := NewClient(
		WithBaseURL("https://example.atlassian.net"),
		WithAssetsCloudID("cloud-1"),
		WithTransport(transport.New()),
	)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	_, err = clientWorkspaceMissing.Assets().GetObject(context.Background(), "1")
	if err == nil || !strings.Contains(err.Error(), "assets workspace ID is required") {
		t.Fatalf("expected workspace ID error, got: %v", err)
	}
}
