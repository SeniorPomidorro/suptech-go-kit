package atlassian

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/SeniorPomidorro/suptech-go-kit/pkg/transport"
)

const defaultAssetsPageSize = 100

// AssetsService provides Jira Assets API operations.
type AssetsService struct {
	client *Client
}

// SearchObjectsAQL searches assets objects using AQL query.
func (s *AssetsService) SearchObjectsAQL(ctx context.Context, aql string, opts *AssetsSearchOptions) (*AssetsSearchResult, error) {
	if strings.TrimSpace(aql) == "" {
		return nil, errors.New("atlassian: aql is required")
	}

	path, err := s.client.assetsPath("/object/aql")
	if err != nil {
		return nil, err
	}

	if opts == nil {
		opts = &AssetsSearchOptions{}
	}

	pageSize := opts.PageSize
	if pageSize <= 0 {
		pageSize = defaultAssetsPageSize
	}

	startAt := opts.StartAt
	result := &AssetsSearchResult{
		StartAt:    opts.StartAt,
		MaxResults: pageSize,
	}

	for {
		payload := map[string]any{
			"qlQuery":    aql,
			"startAt":    startAt,
			"maxResults": pageSize,
		}
		if opts.IncludeAttributes {
			payload["includeAttributes"] = true
		}
		if opts.IncludeTypeAttributes {
			payload["includeTypeAttributes"] = true
		}

		req, err := s.client.newCloudRequest(ctx, http.MethodPost, path, nil, payload)
		if err != nil {
			return nil, err
		}

		var page AssetsSearchResult
		if err := s.client.transport.DoJSON(req, &page); err != nil {
			return nil, err
		}
		objects := page.objects()

		if !opts.FetchAll {
			if len(page.Values) == 0 {
				page.Values = objects
			}
			return &page, nil
		}

		result.Values = append(result.Values, objects...)
		result.Total = page.Total
		result.IsLast = page.IsLast
		startAt += len(objects)
		if page.IsLast || len(objects) == 0 || (page.Total > 0 && startAt >= page.Total) {
			return result, nil
		}
	}
}

// CreateObject creates a Jira Assets object.
func (s *AssetsService) CreateObject(ctx context.Context, payload *CreateAssetObjectRequest) (*AssetObject, error) {
	if payload == nil {
		return nil, errors.New("atlassian: create object payload is required")
	}
	if strings.TrimSpace(payload.ObjectTypeID) == "" {
		return nil, errors.New("atlassian: objectTypeId is required")
	}

	path, err := s.client.assetsPath("/object/create")
	if err != nil {
		return nil, err
	}

	req, err := s.client.newCloudRequest(ctx, http.MethodPost, path, nil, payload)
	if err != nil {
		return nil, err
	}

	var object AssetObject
	if err := s.client.transport.DoJSON(req, &object); err != nil {
		return nil, err
	}
	return &object, nil
}

// DeleteObject removes Jira Assets object by ID.
func (s *AssetsService) DeleteObject(ctx context.Context, objectID string) error {
	if strings.TrimSpace(objectID) == "" {
		return errors.New("atlassian: object ID is required")
	}

	path, err := s.client.assetsPath("/object/" + url.PathEscape(objectID))
	if err != nil {
		return err
	}

	req, err := s.client.newCloudRequest(ctx, http.MethodDelete, path, nil, nil)
	if err != nil {
		return err
	}
	return s.client.doNoResponseBody(req)
}

// UpdateObject updates Jira Assets object.
func (s *AssetsService) UpdateObject(ctx context.Context, objectID string, payload *UpdateAssetObjectRequest) (*AssetObject, error) {
	if strings.TrimSpace(objectID) == "" {
		return nil, errors.New("atlassian: object ID is required")
	}
	if payload == nil {
		return nil, errors.New("atlassian: update object payload is required")
	}

	path, err := s.client.assetsPath("/object/" + url.PathEscape(objectID))
	if err != nil {
		return nil, err
	}

	req, err := s.client.newCloudRequest(ctx, http.MethodPut, path, nil, payload)
	if err != nil {
		return nil, err
	}

	resp, err := s.client.transport.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, transport.NewAPIError(resp, 0)
	}

	var object AssetObject
	if err := json.NewDecoder(resp.Body).Decode(&object); err != nil {
		if errors.Is(err, io.EOF) || resp.StatusCode == http.StatusNoContent {
			return &AssetObject{ID: objectID}, nil
		}
		return nil, fmt.Errorf("atlassian: decode updated object response: %w", err)
	}
	if object.ID == "" {
		object.ID = objectID
	}
	return &object, nil
}

// GetObject fetches Jira Assets object by ID.
func (s *AssetsService) GetObject(ctx context.Context, objectID string) (*AssetObject, error) {
	if strings.TrimSpace(objectID) == "" {
		return nil, errors.New("atlassian: object ID is required")
	}

	path, err := s.client.assetsPath("/object/" + url.PathEscape(objectID))
	if err != nil {
		return nil, err
	}

	req, err := s.client.newCloudRequest(ctx, http.MethodGet, path, nil, nil)
	if err != nil {
		return nil, err
	}

	var object AssetObject
	if err := s.client.transport.DoJSON(req, &object); err != nil {
		return nil, err
	}
	return &object, nil
}

// GetObjectSchema fetches a Jira Assets object schema by ID.
func (s *AssetsService) GetObjectSchema(ctx context.Context, schemaID string) (*ObjectSchema, error) {
	if strings.TrimSpace(schemaID) == "" {
		return nil, errors.New("atlassian: schema ID is required")
	}

	path, err := s.client.assetsPath("/objectschema/" + url.PathEscape(schemaID))
	if err != nil {
		return nil, err
	}

	req, err := s.client.newCloudRequest(ctx, http.MethodGet, path, nil, nil)
	if err != nil {
		return nil, err
	}

	var schema ObjectSchema
	if err := s.client.transport.DoJSON(req, &schema); err != nil {
		return nil, err
	}
	return &schema, nil
}

// GetSchemaObjectTypes returns a flat list of object types for the given schema.
func (s *AssetsService) GetSchemaObjectTypes(ctx context.Context, schemaID string) ([]ObjectTypeEntry, error) {
	if strings.TrimSpace(schemaID) == "" {
		return nil, errors.New("atlassian: schema ID is required")
	}

	path, err := s.client.assetsPath("/objectschema/" + url.PathEscape(schemaID) + "/objecttypes/flat")
	if err != nil {
		return nil, err
	}

	req, err := s.client.newCloudRequest(ctx, http.MethodGet, path, nil, nil)
	if err != nil {
		return nil, err
	}

	var entries []ObjectTypeEntry
	if err := s.client.transport.DoJSON(req, &entries); err != nil {
		return nil, err
	}
	return entries, nil
}

// ListObjectSchemas returns all object schemas visible to the caller.
func (s *AssetsService) ListObjectSchemas(ctx context.Context) (*ObjectSchemaList, error) {
	path, err := s.client.assetsPath("/objectschema/list")
	if err != nil {
		return nil, err
	}

	req, err := s.client.newCloudRequest(ctx, http.MethodGet, path, nil, nil)
	if err != nil {
		return nil, err
	}

	var list ObjectSchemaList
	if err := s.client.transport.DoJSON(req, &list); err != nil {
		return nil, err
	}
	return &list, nil
}

// GetObjectTypeAttributes returns attribute definitions for the given object type.
func (s *AssetsService) GetObjectTypeAttributes(ctx context.Context, objectTypeID string) ([]ObjectTypeAttribute, error) {
	if strings.TrimSpace(objectTypeID) == "" {
		return nil, errors.New("atlassian: object type ID is required")
	}

	path, err := s.client.assetsPath("/objecttype/" + url.PathEscape(objectTypeID) + "/attributes")
	if err != nil {
		return nil, err
	}

	req, err := s.client.newCloudRequest(ctx, http.MethodGet, path, nil, nil)
	if err != nil {
		return nil, err
	}

	var attrs []ObjectTypeAttribute
	if err := s.client.transport.DoJSON(req, &attrs); err != nil {
		return nil, err
	}
	return attrs, nil
}

func (c *Client) assetsPath(pathSuffix string) (string, error) {
	if strings.TrimSpace(c.assetsCloudID) == "" {
		return "", errors.New("atlassian: assets cloud ID is required")
	}
	if strings.TrimSpace(c.assetsWorkspaceID) == "" {
		return "", errors.New("atlassian: assets workspace ID is required")
	}

	base := "/ex/jira/" + url.PathEscape(c.assetsCloudID) + "/jsm/assets/workspace/" + url.PathEscape(c.assetsWorkspaceID) + "/v1"
	if pathSuffix == "" {
		return base, nil
	}
	if strings.HasPrefix(pathSuffix, "/") {
		return base + pathSuffix, nil
	}
	return base + "/" + pathSuffix, nil
}
