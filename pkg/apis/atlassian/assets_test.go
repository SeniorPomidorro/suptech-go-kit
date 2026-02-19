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

func TestCreateObject(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/ex/jira/cloud-1/jsm/assets/workspace/ws-1/v1/object/create" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		var payload CreateAssetObjectRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}

		if payload.ObjectTypeID != "123" {
			t.Fatalf("unexpected objectTypeId: %q", payload.ObjectTypeID)
		}
		if len(payload.Attributes) != 2 {
			t.Fatalf("expected 2 attributes, got %d", len(payload.Attributes))
		}
		if payload.Attributes[0].ObjectTypeAttributeID != "135" {
			t.Fatalf("unexpected first attribute ID: %q", payload.Attributes[0].ObjectTypeAttributeID)
		}
		if len(payload.Attributes[0].ObjectAttributeValues) != 1 {
			t.Fatalf("expected 1 value in first attribute, got %d", len(payload.Attributes[0].ObjectAttributeValues))
		}
		if payload.Attributes[0].ObjectAttributeValues[0].Value != "NY-1" {
			t.Fatalf("unexpected value: %q", payload.Attributes[0].ObjectAttributeValues[0].Value)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"999","objectKey":"OBJ-999","label":"Created Object"}`))
	}))
	defer srv.Close()

	client, err := NewClient(
		WithBaseURL(srv.URL),
		WithCloudBaseURL(srv.URL),
		WithAssetsCloudID("cloud-1"),
		WithAssetsWorkspaceID("ws-1"),
		WithTransport(transport.New()),
	)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	payload := &CreateAssetObjectRequest{
		ObjectTypeID: "123",
		Attributes: []CreateAssetObjectAttribute{
			{
				ObjectTypeAttributeID: "135",
				ObjectAttributeValues: []CreateAssetAttributeValue{
					{Value: "NY-1"},
				},
			},
			{
				ObjectTypeAttributeID: "144",
				ObjectAttributeValues: []CreateAssetAttributeValue{
					{Value: "99"},
				},
			},
		},
	}

	object, err := client.Assets().CreateObject(context.Background(), payload)
	if err != nil {
		t.Fatalf("CreateObject failed: %v", err)
	}
	if object.ID != "999" {
		t.Fatalf("unexpected object ID: %q", object.ID)
	}
	if object.ObjectKey != "OBJ-999" {
		t.Fatalf("unexpected object key: %q", object.ObjectKey)
	}
}

func TestCreateObjectValidation(t *testing.T) {
	t.Parallel()

	client, err := NewClient(
		WithBaseURL("https://example.atlassian.net"),
		WithAssetsCloudID("cloud-1"),
		WithAssetsWorkspaceID("ws-1"),
		WithTransport(transport.New()),
	)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	t.Run("nil payload", func(t *testing.T) {
		_, err := client.Assets().CreateObject(context.Background(), nil)
		if err == nil || !strings.Contains(err.Error(), "payload is required") {
			t.Fatalf("expected payload error, got: %v", err)
		}
	})

	t.Run("empty objectTypeId", func(t *testing.T) {
		_, err := client.Assets().CreateObject(context.Background(), &CreateAssetObjectRequest{})
		if err == nil || !strings.Contains(err.Error(), "objectTypeId is required") {
			t.Fatalf("expected objectTypeId error, got: %v", err)
		}
	})
}

func TestUpdateObject(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/ex/jira/cloud-1/jsm/assets/workspace/ws-1/v1/object/42" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}

		var payload UpdateAssetObjectRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode payload: %v", err)
		}

		if len(payload.Attributes) != 1 {
			t.Fatalf("expected 1 attribute, got %d", len(payload.Attributes))
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"42","objectKey":"OBJ-42","label":"Updated Object"}`))
	}))
	defer srv.Close()

	client, err := NewClient(
		WithBaseURL(srv.URL),
		WithCloudBaseURL(srv.URL),
		WithAssetsCloudID("cloud-1"),
		WithAssetsWorkspaceID("ws-1"),
		WithTransport(transport.New()),
	)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	payload := &UpdateAssetObjectRequest{
		Attributes: []CreateAssetObjectAttribute{
			{
				ObjectTypeAttributeID: "135",
				ObjectAttributeValues: []CreateAssetAttributeValue{
					{Value: "NY-2"},
				},
			},
		},
	}

	object, err := client.Assets().UpdateObject(context.Background(), "42", payload)
	if err != nil {
		t.Fatalf("UpdateObject failed: %v", err)
	}
	if object.ID != "42" {
		t.Fatalf("unexpected object ID: %q", object.ID)
	}
}

func TestUpdateObjectValidation(t *testing.T) {
	t.Parallel()

	client, err := NewClient(
		WithBaseURL("https://example.atlassian.net"),
		WithAssetsCloudID("cloud-1"),
		WithAssetsWorkspaceID("ws-1"),
		WithTransport(transport.New()),
	)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	t.Run("empty objectId", func(t *testing.T) {
		_, err := client.Assets().UpdateObject(context.Background(), "", &UpdateAssetObjectRequest{})
		if err == nil || !strings.Contains(err.Error(), "object ID is required") {
			t.Fatalf("expected object ID error, got: %v", err)
		}
	})

	t.Run("nil payload", func(t *testing.T) {
		_, err := client.Assets().UpdateObject(context.Background(), "42", nil)
		if err == nil || !strings.Contains(err.Error(), "payload is required") {
			t.Fatalf("expected payload error, got: %v", err)
		}
	})
}

func TestDeleteObject(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/ex/jira/cloud-1/jsm/assets/workspace/ws-1/v1/object/42" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	client, err := NewClient(
		WithBaseURL(srv.URL),
		WithCloudBaseURL(srv.URL),
		WithAssetsCloudID("cloud-1"),
		WithAssetsWorkspaceID("ws-1"),
		WithTransport(transport.New()),
	)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	err = client.Assets().DeleteObject(context.Background(), "42")
	if err != nil {
		t.Fatalf("DeleteObject failed: %v", err)
	}
}

func TestNewCreateAssetObjectRequest(t *testing.T) {
	t.Parallel()

	input := AssetObjectInput{
		ObjectTypeID: "123",
		Attributes: []AssetAttributeInput{
			{
				ObjectTypeAttributeID: "135",
				Values:                []string{"NY-1", "NY-2"},
			},
			{
				ObjectTypeAttributeID: "144",
				Values:                []string{"99"},
			},
		},
	}

	req := NewCreateAssetObjectRequest(input)

	if req.ObjectTypeID != "123" {
		t.Fatalf("unexpected objectTypeId: %q", req.ObjectTypeID)
	}
	if len(req.Attributes) != 2 {
		t.Fatalf("expected 2 attributes, got %d", len(req.Attributes))
	}
	if req.Attributes[0].ObjectTypeAttributeID != "135" {
		t.Fatalf("unexpected first attribute ID: %q", req.Attributes[0].ObjectTypeAttributeID)
	}
	if len(req.Attributes[0].ObjectAttributeValues) != 2 {
		t.Fatalf("expected 2 values in first attribute, got %d", len(req.Attributes[0].ObjectAttributeValues))
	}
	if req.Attributes[0].ObjectAttributeValues[0].Value != "NY-1" {
		t.Fatalf("unexpected first value: %q", req.Attributes[0].ObjectAttributeValues[0].Value)
	}
	if req.Attributes[0].ObjectAttributeValues[1].Value != "NY-2" {
		t.Fatalf("unexpected second value: %q", req.Attributes[0].ObjectAttributeValues[1].Value)
	}
	if len(req.Attributes[1].ObjectAttributeValues) != 1 {
		t.Fatalf("expected 1 value in second attribute, got %d", len(req.Attributes[1].ObjectAttributeValues))
	}
}

func TestGetSchemaObjectTypes(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		wantPath := "/ex/jira/cloud-1/jsm/assets/workspace/ws-1/v1/objectschema/6/objecttypes/flat"
		if r.URL.Path != wantPath {
			t.Fatalf("unexpected path: got=%s want=%s", r.URL.Path, wantPath)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{"id":"1","name":"Server","objectCount":42,"position":0},
			{"id":"2","name":"Database","objectCount":15,"position":1},
			{"id":"3","name":"Network Device","objectCount":7,"position":2}
		]`))
	}))
	defer srv.Close()

	client, err := NewClient(
		WithBaseURL(srv.URL),
		WithCloudBaseURL(srv.URL),
		WithAssetsCloudID("cloud-1"),
		WithAssetsWorkspaceID("ws-1"),
		WithTransport(transport.New()),
	)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	entries, err := client.Assets().GetSchemaObjectTypes(context.Background(), "6")
	if err != nil {
		t.Fatalf("GetSchemaObjectTypes failed: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}
	if entries[0].ID != "1" || entries[0].Name != "Server" || entries[0].ObjectCount != 42 {
		t.Fatalf("unexpected first entry: %+v", entries[0])
	}
	if entries[1].ID != "2" || entries[1].Name != "Database" || entries[1].ObjectCount != 15 {
		t.Fatalf("unexpected second entry: %+v", entries[1])
	}
	if entries[2].Position != 2 {
		t.Fatalf("unexpected third entry position: %d", entries[2].Position)
	}
}

func TestGetSchemaObjectTypesValidation(t *testing.T) {
	t.Parallel()

	client, err := NewClient(
		WithBaseURL("https://example.atlassian.net"),
		WithAssetsCloudID("cloud-1"),
		WithAssetsWorkspaceID("ws-1"),
		WithTransport(transport.New()),
	)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	_, err = client.Assets().GetSchemaObjectTypes(context.Background(), "")
	if err == nil || !strings.Contains(err.Error(), "schema ID is required") {
		t.Fatalf("expected schema ID error, got: %v", err)
	}
}

func TestListObjectSchemas(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		wantPath := "/ex/jira/cloud-1/jsm/assets/workspace/ws-1/v1/objectschema/list"
		if r.URL.Path != wantPath {
			t.Fatalf("unexpected path: got=%s want=%s", r.URL.Path, wantPath)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"values":[
			{"id":"6","name":"IT Assets","objectSchemaKey":"ITA","objectCount":100,"objectTypeCount":5},
			{"id":"7","name":"HR Assets","objectSchemaKey":"HRA","objectCount":50,"objectTypeCount":3}
		]}`))
	}))
	defer srv.Close()

	client, err := NewClient(
		WithBaseURL(srv.URL),
		WithCloudBaseURL(srv.URL),
		WithAssetsCloudID("cloud-1"),
		WithAssetsWorkspaceID("ws-1"),
		WithTransport(transport.New()),
	)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	list, err := client.Assets().ListObjectSchemas(context.Background())
	if err != nil {
		t.Fatalf("ListObjectSchemas failed: %v", err)
	}
	if len(list.Values) != 2 {
		t.Fatalf("expected 2 schemas, got %d", len(list.Values))
	}
	if list.Values[0].ID != "6" || list.Values[0].Name != "IT Assets" {
		t.Fatalf("unexpected first schema: %+v", list.Values[0])
	}
	if list.Values[0].ObjectSchemaKey != "ITA" {
		t.Fatalf("unexpected schema key: %q", list.Values[0].ObjectSchemaKey)
	}
	if list.Values[0].ObjectCount != 100 || list.Values[0].ObjectTypeCount != 5 {
		t.Fatalf("unexpected counts: objectCount=%d objectTypeCount=%d", list.Values[0].ObjectCount, list.Values[0].ObjectTypeCount)
	}
	if list.Values[1].ID != "7" || list.Values[1].Name != "HR Assets" {
		t.Fatalf("unexpected second schema: %+v", list.Values[1])
	}
}

func TestGetObjectTypeAttributes(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		wantPath := "/ex/jira/cloud-1/jsm/assets/workspace/ws-1/v1/objecttype/42/attributes"
		if r.URL.Path != wantPath {
			t.Fatalf("unexpected path: got=%s want=%s", r.URL.Path, wantPath)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[
			{"id":"1","name":"Name","type":0,"defaultType":{"id":0,"name":"Text"},"editable":true,"system":true,"position":0},
			{"id":"2","name":"Owner","type":2,"editable":true,"system":false,"position":1},
			{"id":"3","name":"Parent Rack","type":1,"editable":true,"referenceObjectTypeId":"99","position":2}
		]`))
	}))
	defer srv.Close()

	client, err := NewClient(
		WithBaseURL(srv.URL),
		WithCloudBaseURL(srv.URL),
		WithAssetsCloudID("cloud-1"),
		WithAssetsWorkspaceID("ws-1"),
		WithTransport(transport.New()),
	)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	attrs, err := client.Assets().GetObjectTypeAttributes(context.Background(), "42")
	if err != nil {
		t.Fatalf("GetObjectTypeAttributes failed: %v", err)
	}
	if len(attrs) != 3 {
		t.Fatalf("expected 3 attributes, got %d", len(attrs))
	}
	if attrs[0].ID != "1" || attrs[0].Name != "Name" || attrs[0].Type != 0 {
		t.Fatalf("unexpected first attr: %+v", attrs[0])
	}
	if attrs[0].DefaultType == nil || attrs[0].DefaultType.ID != 0 || attrs[0].DefaultType.Name != "Text" {
		t.Fatalf("unexpected first attr default type: %+v", attrs[0].DefaultType)
	}
	if attrs[1].ID != "2" || attrs[1].Name != "Owner" || attrs[1].Type != 2 {
		t.Fatalf("unexpected second attr: %+v", attrs[1])
	}
	if attrs[2].ReferenceObjectTypeID != "99" {
		t.Fatalf("unexpected reference object type ID: %q", attrs[2].ReferenceObjectTypeID)
	}
}

func TestGetObjectTypeAttributesValidation(t *testing.T) {
	t.Parallel()

	client, err := NewClient(
		WithBaseURL("https://example.atlassian.net"),
		WithAssetsCloudID("cloud-1"),
		WithAssetsWorkspaceID("ws-1"),
		WithTransport(transport.New()),
	)
	if err != nil {
		t.Fatalf("new client: %v", err)
	}

	_, err = client.Assets().GetObjectTypeAttributes(context.Background(), "")
	if err == nil || !strings.Contains(err.Error(), "object type ID is required") {
		t.Fatalf("expected object type ID error, got: %v", err)
	}
}

func TestNewUpdateAssetObjectRequest(t *testing.T) {
	t.Parallel()

	input := AssetObjectInput{
		ObjectTypeID: "456",
		Attributes: []AssetAttributeInput{
			{
				ObjectTypeAttributeID: "200",
				Values:                []string{"Updated Value"},
			},
		},
	}

	req := NewUpdateAssetObjectRequest(input)

	if req.ObjectTypeID != "456" {
		t.Fatalf("unexpected objectTypeId: %q", req.ObjectTypeID)
	}
	if len(req.Attributes) != 1 {
		t.Fatalf("expected 1 attribute, got %d", len(req.Attributes))
	}
	if req.Attributes[0].ObjectTypeAttributeID != "200" {
		t.Fatalf("unexpected attribute ID: %q", req.Attributes[0].ObjectTypeAttributeID)
	}
	if len(req.Attributes[0].ObjectAttributeValues) != 1 {
		t.Fatalf("expected 1 value, got %d", len(req.Attributes[0].ObjectAttributeValues))
	}
	if req.Attributes[0].ObjectAttributeValues[0].Value != "Updated Value" {
		t.Fatalf("unexpected value: %q", req.Attributes[0].ObjectAttributeValues[0].Value)
	}
}
