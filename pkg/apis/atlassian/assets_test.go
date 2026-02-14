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

	result, err := client.Assets().SearchObjectsAQL(context.Background(), "Name LIKE server*", &AssetsSearchOptions{
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

func TestAssetsSearchResultFindMethods(t *testing.T) {
	t.Parallel()

	result := &AssetsSearchResult{
		Values: []AssetObject{
			{ID: "1", ObjectKey: "KEY-1", Label: "Server 1"},
			{ID: "2", ObjectKey: "KEY-2", Label: "Server 2"},
			{ID: "3", ObjectKey: "KEY-3", Label: "Database"},
		},
	}

	t.Run("FindObjectByID", func(t *testing.T) {
		obj := result.FindObjectByID("2")
		if obj == nil {
			t.Fatal("expected to find object")
		}
		if obj.ID != "2" || obj.Label != "Server 2" {
			t.Errorf("unexpected object: %+v", obj)
		}

		notFound := result.FindObjectByID("999")
		if notFound != nil {
			t.Error("expected nil for non-existent ID")
		}
	})

	t.Run("FindObjectByKey", func(t *testing.T) {
		obj := result.FindObjectByKey("KEY-3")
		if obj == nil {
			t.Fatal("expected to find object")
		}
		if obj.ObjectKey != "KEY-3" || obj.Label != "Database" {
			t.Errorf("unexpected object: %+v", obj)
		}

		notFound := result.FindObjectByKey("KEY-999")
		if notFound != nil {
			t.Error("expected nil for non-existent key")
		}
	})

	t.Run("FindObjectByLabel", func(t *testing.T) {
		obj := result.FindObjectByLabel("Server 1")
		if obj == nil {
			t.Fatal("expected to find object")
		}
		if obj.Label != "Server 1" || obj.ID != "1" {
			t.Errorf("unexpected object: %+v", obj)
		}

		notFound := result.FindObjectByLabel("NonExistent")
		if notFound != nil {
			t.Error("expected nil for non-existent label")
		}
	})

	t.Run("ObjectEntries fallback", func(t *testing.T) {
		resultWithEntries := &AssetsSearchResult{
			ObjectEntries: []AssetObject{
				{ID: "10", ObjectKey: "KEY-10", Label: "Entry 1"},
			},
		}

		obj := resultWithEntries.FindObjectByID("10")
		if obj == nil || obj.ID != "10" {
			t.Error("expected to find object in ObjectEntries")
		}
	})
}

func TestAssetObjectAttributeMethods(t *testing.T) {
	t.Parallel()

	obj := &AssetObject{
		ID:    "911",
		Label: "Test Object",
		Attributes: []AssetObjectAttr{
			{
				ObjectTypeAttributeID: "402",
				ObjectAttributeValues: []AssetAttributeValue{
					{DisplayValue: "ServiceDesk Team", Value: "ServiceDesk Team"},
				},
			},
			{
				ObjectTypeAttributeID: "454",
				ObjectAttributeValues: []AssetAttributeValue{
					{
						DisplayValue: "Sergey Rostovskiy",
						SearchValue:  "61f983b2845d670071f2c97d",
						User: &AssetAttributeUser{
							DisplayName:  "Sergey Rostovskiy",
							EmailAddress: "sergey.rostovskiy@tabby.ai",
							Key:          "61f983b2845d670071f2c97d",
						},
					},
				},
			},
			{
				ObjectTypeAttributeID: "449",
				ObjectAttributeValues: []AssetAttributeValue{
					{DisplayValue: "Value 1", Value: "v1"},
					{DisplayValue: "Value 2", Value: "v2"},
				},
			},
		},
	}

	t.Run("GetAttributeByID", func(t *testing.T) {
		attr := obj.GetAttributeByID("402")
		if attr == nil {
			t.Fatal("expected to find attribute")
		}
		if attr.ObjectTypeAttributeID != "402" {
			t.Errorf("unexpected attribute ID: %s", attr.ObjectTypeAttributeID)
		}
		if len(attr.ObjectAttributeValues) != 1 {
			t.Errorf("expected 1 value, got %d", len(attr.ObjectAttributeValues))
		}

		notFound := obj.GetAttributeByID("999")
		if notFound != nil {
			t.Error("expected nil for non-existent attribute ID")
		}
	})

	t.Run("GetAttributeValues", func(t *testing.T) {
		values := obj.GetAttributeValues("449")
		if len(values) != 2 {
			t.Fatalf("expected 2 values, got %d", len(values))
		}
		if values[0].DisplayValue != "Value 1" || values[1].DisplayValue != "Value 2" {
			t.Errorf("unexpected values: %+v", values)
		}

		notFound := obj.GetAttributeValues("999")
		if notFound != nil {
			t.Error("expected nil for non-existent attribute ID")
		}
	})

	t.Run("GetAttributeValues with User", func(t *testing.T) {
		values := obj.GetAttributeValues("454")
		if len(values) != 1 {
			t.Fatalf("expected 1 value, got %d", len(values))
		}
		if values[0].User == nil {
			t.Fatal("expected user to be present")
		}
		if values[0].User.EmailAddress != "sergey.rostovskiy@tabby.ai" {
			t.Errorf("unexpected email: %s", values[0].User.EmailAddress)
		}
	})
}
