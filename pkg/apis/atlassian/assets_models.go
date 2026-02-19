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

// ObjectSchema represents a Jira Assets object schema.
type ObjectSchema struct {
	WorkspaceID     string `json:"workspaceId,omitempty"`
	GlobalID        string `json:"globalId,omitempty"`
	ID              string `json:"id"`
	Name            string `json:"name"`
	ObjectSchemaKey string `json:"objectSchemaKey"`
	Status          string `json:"status,omitempty"`
	Description     string `json:"description,omitempty"`
	Created         string `json:"created,omitempty"`
	Updated         string `json:"updated,omitempty"`
	ObjectCount     int    `json:"objectCount,omitempty"`
	ObjectTypeCount int    `json:"objectTypeCount,omitempty"`
}

// ObjectTypeEntry represents a single object type within a schema.
type ObjectTypeEntry struct {
	WorkspaceID        string `json:"workspaceId,omitempty"`
	GlobalID           string `json:"globalId,omitempty"`
	ID                 string `json:"id"`
	Name               string `json:"name"`
	Description        string `json:"description,omitempty"`
	IconID             string `json:"iconId,omitempty"`
	Position           int    `json:"position"`
	ObjectCount        int    `json:"objectCount"`
	ObjectSchemaID     string `json:"objectSchemaId,omitempty"`
	Inherited          bool   `json:"inherited"`
	AbstractObjectType bool   `json:"abstractObjectType"`
	ParentObjectTypeID string `json:"parentObjectTypeId,omitempty"`
}

// ObjectSchemaList represents a paginated list of object schemas.
type ObjectSchemaList struct {
	Values []ObjectSchema `json:"values"`
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

// CreateAssetObjectRequest represents the payload for creating an asset object.
type CreateAssetObjectRequest struct {
	ObjectTypeID string                        `json:"objectTypeId"`
	Attributes   []CreateAssetObjectAttribute  `json:"attributes"`
}

// CreateAssetObjectAttribute represents a single attribute in the create request.
type CreateAssetObjectAttribute struct {
	ObjectTypeAttributeID string                      `json:"objectTypeAttributeId"`
	ObjectAttributeValues []CreateAssetAttributeValue `json:"objectAttributeValues"`
}

// CreateAssetAttributeValue represents a single value in the create request.
type CreateAssetAttributeValue struct {
	Value string `json:"value"`
}

// UpdateAssetObjectRequest represents the payload for updating an asset object.
type UpdateAssetObjectRequest struct {
	ObjectTypeID string                        `json:"objectTypeId,omitempty"`
	Attributes   []CreateAssetObjectAttribute  `json:"attributes,omitempty"`
}

// AssetObjectInput is a simplified input for building CreateAssetObjectRequest or UpdateAssetObjectRequest.
type AssetObjectInput struct {
	ObjectTypeID string
	Attributes   []AssetAttributeInput
}

// AssetAttributeInput is a simplified attribute input with string slice values.
type AssetAttributeInput struct {
	ObjectTypeAttributeID string
	Values                []string
}

// NewCreateAssetObjectRequest builds a CreateAssetObjectRequest from simplified input.
func NewCreateAssetObjectRequest(input AssetObjectInput) *CreateAssetObjectRequest {
	req := &CreateAssetObjectRequest{
		ObjectTypeID: input.ObjectTypeID,
		Attributes:   make([]CreateAssetObjectAttribute, 0, len(input.Attributes)),
	}

	for _, attr := range input.Attributes {
		values := make([]CreateAssetAttributeValue, 0, len(attr.Values))
		for _, v := range attr.Values {
			values = append(values, CreateAssetAttributeValue{Value: v})
		}
		req.Attributes = append(req.Attributes, CreateAssetObjectAttribute{
			ObjectTypeAttributeID: attr.ObjectTypeAttributeID,
			ObjectAttributeValues: values,
		})
	}

	return req
}

// ObjectTypeAttribute represents an attribute definition for an object type.
type ObjectTypeAttribute struct {
	WorkspaceID             string         `json:"workspaceId,omitempty"`
	GlobalID                string         `json:"globalId,omitempty"`
	ID                      string         `json:"id"`
	Name                    string         `json:"name"`
	Label                   bool           `json:"label"`
	Description             string         `json:"description,omitempty"`
	Type                    int            `json:"type"`
	DefaultType             *AttributeType `json:"defaultType,omitempty"`
	Editable                bool           `json:"editable"`
	System                  bool           `json:"system"`
	Sortable                bool           `json:"sortable"`
	Summable                bool           `json:"summable"`
	Indexed                 bool           `json:"indexed"`
	MinimumCardinality      int            `json:"minimumCardinality"`
	MaximumCardinality      int            `json:"maximumCardinality"`
	Removable               bool           `json:"removable"`
	Hidden                  bool           `json:"hidden"`
	IncludeChildObjectTypes bool           `json:"includeChildObjectTypes"`
	UniqueAttribute         bool           `json:"uniqueAttribute"`
	Options                 string         `json:"options,omitempty"`
	Position                int            `json:"position"`
	ReferenceObjectTypeID   string         `json:"referenceObjectTypeId,omitempty"`
	ReferenceType           *ReferenceType `json:"referenceType,omitempty"`
}

// AttributeType represents the default sub-type of an attribute (for Type=0).
type AttributeType struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// ReferenceType represents the reference type metadata of an object-reference attribute.
type ReferenceType struct {
	WorkspaceID string `json:"workspaceId,omitempty"`
	GlobalID    string `json:"globalId,omitempty"`
	Name        string `json:"name"`
}

// NewUpdateAssetObjectRequest builds an UpdateAssetObjectRequest from simplified input.
func NewUpdateAssetObjectRequest(input AssetObjectInput) *UpdateAssetObjectRequest {
	req := &UpdateAssetObjectRequest{
		ObjectTypeID: input.ObjectTypeID,
		Attributes:   make([]CreateAssetObjectAttribute, 0, len(input.Attributes)),
	}

	for _, attr := range input.Attributes {
		values := make([]CreateAssetAttributeValue, 0, len(attr.Values))
		for _, v := range attr.Values {
			values = append(values, CreateAssetAttributeValue{Value: v})
		}
		req.Attributes = append(req.Attributes, CreateAssetObjectAttribute{
			ObjectTypeAttributeID: attr.ObjectTypeAttributeID,
			ObjectAttributeValues: values,
		})
	}

	return req
}
