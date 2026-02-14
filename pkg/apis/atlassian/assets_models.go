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
	WorkspaceID           string                  `json:"workspaceId,omitempty"`
	GlobalID              string                  `json:"globalId,omitempty"`
	ID                    string                  `json:"id,omitempty"`
	ObjectTypeAttributeID string                  `json:"objectTypeAttributeId,omitempty"`
	ObjectAttributeValues []AssetAttributeValue   `json:"objectAttributeValues,omitempty"`
	ObjectID              string                  `json:"objectId,omitempty"`
	Meta                  map[string]any          `json:"meta,omitempty"`
	Value                 map[string]any          `json:"value,omitempty"`
	Values                []map[string]any        `json:"values,omitempty"`
	Additional            map[string][]string     `json:"additional,omitempty"`
}

// AssetAttributeValue represents a single value within an object attribute.
type AssetAttributeValue struct {
	DisplayValue     string                 `json:"displayValue,omitempty"`
	SearchValue      string                 `json:"searchValue,omitempty"`
	Value            string                 `json:"value,omitempty"`
	ReferencedType   bool                   `json:"referencedType,omitempty"`
	User             *AssetAttributeUser    `json:"user,omitempty"`
	Group            *AssetAttributeGroup   `json:"group,omitempty"`
	ReferencedObject *AssetReferencedObject `json:"referencedObject,omitempty"`
}

// AssetAttributeUser represents a user value in an attribute.
type AssetAttributeUser struct {
	AvatarURL    string `json:"avatarUrl,omitempty"`
	DisplayName  string `json:"displayName,omitempty"`
	Name         string `json:"name,omitempty"`
	Key          string `json:"key,omitempty"`
	EmailAddress string `json:"emailAddress,omitempty"`
	IsDeleted    bool   `json:"isDeleted,omitempty"`
}

// AssetAttributeGroup represents a group value in an attribute.
type AssetAttributeGroup struct {
	AvatarURL string `json:"avatarUrl,omitempty"`
	Name      string `json:"name,omitempty"`
}

// AssetReferencedObject represents a referenced asset object in an attribute.
type AssetReferencedObject struct {
	ID         string           `json:"id,omitempty"`
	ObjectKey  string           `json:"objectKey,omitempty"`
	Label      string           `json:"label,omitempty"`
	ObjectType *AssetObjectType `json:"objectType,omitempty"`
}

func (r AssetsSearchResult) objects() []AssetObject {
	if len(r.Values) > 0 {
		return r.Values
	}
	return r.ObjectEntries
}

// FindObjectByID returns an object by its ID.
func (r *AssetsSearchResult) FindObjectByID(id string) *AssetObject {
	objects := r.objects()
	for i := range objects {
		if objects[i].ID == id {
			return &objects[i]
		}
	}
	return nil
}

// FindObjectByKey returns an object by its ObjectKey.
func (r *AssetsSearchResult) FindObjectByKey(key string) *AssetObject {
	objects := r.objects()
	for i := range objects {
		if objects[i].ObjectKey == key {
			return &objects[i]
		}
	}
	return nil
}

// FindObjectByLabel returns an object by its Label.
func (r *AssetsSearchResult) FindObjectByLabel(label string) *AssetObject {
	objects := r.objects()
	for i := range objects {
		if objects[i].Label == label {
			return &objects[i]
		}
	}
	return nil
}

// GetAttributeByID returns an attribute by its ObjectTypeAttributeID.
func (o *AssetObject) GetAttributeByID(attributeID string) *AssetObjectAttr {
	for i := range o.Attributes {
		if o.Attributes[i].ObjectTypeAttributeID == attributeID {
			return &o.Attributes[i]
		}
	}
	return nil
}

// GetAttributeValues returns all values of an attribute by its ObjectTypeAttributeID.
func (o *AssetObject) GetAttributeValues(attributeID string) []AssetAttributeValue {
	attr := o.GetAttributeByID(attributeID)
	if attr == nil {
		return nil
	}
	return attr.ObjectAttributeValues
}
