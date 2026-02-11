package atlassian

// AssetsSearchOptions controls AQL pagination and response shape.
type AssetsSearchOptions struct {
	StartAt               int
	PageSize              int
	FetchAll              bool
	IncludeAttributes     bool
	IncludeTypeAttributes bool
}

// AssetsSearchResult is a paginated Assets AQL response.
type AssetsSearchResult struct {
	StartAt       int           `json:"startAt"`
	MaxResults    int           `json:"maxResults"`
	Total         int           `json:"total"`
	IsLast        bool          `json:"isLast"`
	Values        []AssetObject `json:"values,omitempty"`
	ObjectEntries []AssetObject `json:"objectEntries,omitempty"`
}

// AssetObject is a minimal Jira Assets object DTO.
type AssetObject struct {
	ID          string            `json:"id"`
	ObjectKey   string            `json:"objectKey,omitempty"`
	Label       string            `json:"label,omitempty"`
	ObjectType  AssetObjectType   `json:"objectType,omitempty"`
	Attributes  []AssetObjectAttr `json:"attributes,omitempty"`
	Avatar      map[string]any    `json:"avatar,omitempty"`
	Timestamps  map[string]any    `json:"timestamps,omitempty"`
	RawMetadata map[string]any    `json:"metadata,omitempty"`
	Raw         map[string]any    `json:"-"`
}

// AssetObjectType is a minimal object type descriptor.
type AssetObjectType struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

// AssetObjectAttr is a minimal object attribute descriptor.
type AssetObjectAttr struct {
	ObjectTypeAttributeID string              `json:"objectTypeAttributeId,omitempty"`
	ObjectAttributeValues []map[string]any    `json:"objectAttributeValues,omitempty"`
	Meta                  map[string]any      `json:"meta,omitempty"`
	Value                 map[string]any      `json:"value,omitempty"`
	Values                []map[string]any    `json:"values,omitempty"`
	Additional            map[string][]string `json:"additional,omitempty"`
}

func (r AssetsSearchResult) objects() []AssetObject {
	if len(r.Values) > 0 {
		return r.Values
	}
	return r.ObjectEntries
}
