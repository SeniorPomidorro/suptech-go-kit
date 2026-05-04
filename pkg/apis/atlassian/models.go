package atlassian

import (
	"encoding/json"
	"time"
)

// Issue is a minimal Jira issue DTO for client consumers.
type Issue struct {
	ID     string          `json:"id"`
	Key    string          `json:"key"`
	Fields json.RawMessage `json:"fields,omitempty"`
}

// SearchResult is Jira search response (POST /rest/api/3/search/jql).
type SearchResult struct {
	Issues        []Issue `json:"issues"`
	NextPageToken string  `json:"nextPageToken,omitempty"`
	IsLast        bool    `json:"isLast,omitempty"`
}

// Comment is a minimal Jira comment DTO.
type Comment struct {
	ID   string          `json:"id"`
	Body json.RawMessage `json:"body,omitempty"`
}

// Attachment describes uploaded Jira attachment.
type Attachment struct {
	ID       string `json:"id"`
	FileName string `json:"filename"`
	Size     int64  `json:"size,omitempty"`
	Content  string `json:"content,omitempty"`
}

// User is a minimal Jira user DTO.
type User struct {
	AccountID   string `json:"accountId"`
	DisplayName string `json:"displayName"`
	Email       string `json:"emailAddress,omitempty"`
	Active      bool   `json:"active"`
}

// BulkGetUsersOptions controls GET /rest/api/3/user/bulk query parameters.
type BulkGetUsersOptions struct {
	StartAt    int
	MaxResults int
	Usernames  []string
	Keys       []string
	AccountIDs []string
	FetchAll   bool
	// FetchAllThrottle defines pause between page requests when FetchAll=true.
	// If <=0, a default of 200 milliseconds is used.
	FetchAllThrottle time.Duration
}

// BulkUsersResult is a paginated response from GET /rest/api/3/user/bulk.
type BulkUsersResult struct {
	Self       string `json:"self,omitempty"`
	NextPage   string `json:"nextPage,omitempty"`
	MaxResults int    `json:"maxResults,omitempty"`
	StartAt    int    `json:"startAt,omitempty"`
	Total      int    `json:"total,omitempty"`
	IsLast     bool   `json:"isLast,omitempty"`
	Values     []User `json:"values,omitempty"`
}

// CreateIssueRequest is the payload for POST /rest/api/3/issue.
type CreateIssueRequest struct {
	Fields          map[string]any   `json:"fields,omitempty"`
	Update          map[string][]any `json:"update,omitempty"`
	Properties      []map[string]any `json:"properties,omitempty"`
	HistoryMetadata map[string]any   `json:"historyMetadata,omitempty"`
}

// CreatedIssue is the response from POST /rest/api/3/issue.
type CreatedIssue struct {
	ID   string `json:"id"`
	Key  string `json:"key"`
	Self string `json:"self,omitempty"`
}

// UpdateIssueRequest is the payload for PUT /rest/api/3/issue/{issueIdOrKey}.
type UpdateIssueRequest struct {
	Fields          map[string]any   `json:"fields,omitempty"`
	Update          map[string][]any `json:"update,omitempty"`
	Properties      []map[string]any `json:"properties,omitempty"`
	HistoryMetadata map[string]any   `json:"historyMetadata,omitempty"`
}

// UpdateIssueOptions controls query parameters for issue update.
type UpdateIssueOptions struct {
	NotifyUsers            *bool
	OverrideScreenSecurity bool
	OverrideEditableFlag   bool
	ReturnIssue            bool
	Expand                 string
}

// TransitionStatus describes the target status of a transition.
type TransitionStatus struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

// Transition describes an available workflow transition on an issue.
type Transition struct {
	ID            string           `json:"id"`
	Name          string           `json:"name,omitempty"`
	To            TransitionStatus `json:"to,omitempty"`
	HasScreen     bool             `json:"hasScreen,omitempty"`
	IsGlobal      bool             `json:"isGlobal,omitempty"`
	IsInitial     bool             `json:"isInitial,omitempty"`
	IsAvailable   bool             `json:"isAvailable,omitempty"`
	IsConditional bool             `json:"isConditional,omitempty"`
	Looped        bool             `json:"looped,omitempty"`

	// Fields is populated only when GetTransitions is invoked with
	// GetTransitionsOptions.Expand containing "transitions.fields". Map key is
	// the Jira field key (system name like "summary" or custom field id like
	// "customfield_12345"). Use it to render a transition screen UI and to
	// build the payload for DoTransition.
	Fields map[string]FieldMetadata `json:"fields,omitempty"`
}

// FieldMetadata describes one editable field on a transition or create screen.
// Returned only when GetTransitions is invoked with Expand="transitions.fields"
// or by createMeta endpoints. Mirrors the Jira Cloud OpenAPI schema FieldMetadata.
type FieldMetadata struct {
	Required        bool                `json:"required"`
	Name            string              `json:"name,omitempty"`
	Key             string              `json:"key,omitempty"`
	Operations      []string            `json:"operations,omitempty"`
	Schema          FieldMetadataSchema `json:"schema,omitempty"`
	AllowedValues   []AllowedValue      `json:"allowedValues,omitempty"`
	HasDefaultValue bool                `json:"hasDefaultValue,omitempty"`
	DefaultValue    json.RawMessage     `json:"defaultValue,omitempty"`
	AutoCompleteURL string              `json:"autoCompleteUrl,omitempty"`
}

// FieldMetadataSchema describes a field's data type. Mirrors the Jira Cloud
// OpenAPI schema JsonTypeBean (renamed for Go readability).
type FieldMetadataSchema struct {
	// Type is the data type of the field, e.g. "string", "number", "date",
	// "datetime", "option", "priority", "user", "array".
	Type string `json:"type,omitempty"`
	// Items names the element type when Type=="array".
	Items string `json:"items,omitempty"`
	// System is the system field name for built-in fields (e.g. "summary",
	// "description", "priority", "assignee").
	System string `json:"system,omitempty"`
	// Custom is the URI of the custom field type.
	Custom string `json:"custom,omitempty"`
	// CustomID is the numeric id of the custom field, e.g. 12345 for
	// customfield_12345.
	CustomID int64 `json:"customId,omitempty"`
}

// AllowedValue is a single value that the field accepts. The Jira API uses a
// shared array shape for allowedValues across many field types, with each
// type populating a different subset of properties:
//   - option / resolution / version / component: ID + Name (some option fields use Value)
//   - priority: ID + Name
//   - user: AccountID + DisplayName + (optional) EmailAddress
//   - group: Name + (optional) GroupID
type AllowedValue struct {
	ID           string `json:"id,omitempty"`
	Name         string `json:"name,omitempty"`
	Value        string `json:"value,omitempty"`
	Description  string `json:"description,omitempty"`
	AccountID    string `json:"accountId,omitempty"`
	DisplayName  string `json:"displayName,omitempty"`
	EmailAddress string `json:"emailAddress,omitempty"`
	GroupID      string `json:"groupId,omitempty"`
}

// TransitionsList is the response of GET /rest/api/3/issue/{issueIdOrKey}/transitions.
type TransitionsList struct {
	Expand      string       `json:"expand,omitempty"`
	Transitions []Transition `json:"transitions"`
}

// GetTransitionsOptions controls query parameters for listing transitions.
type GetTransitionsOptions struct {
	Expand                        string
	TransitionID                  string
	SkipRemoteOnlyCondition       bool
	IncludeUnavailableTransitions bool
	SortByOpsBarAndStatus         bool
}

// DoTransitionRequest is the payload for POST /rest/api/3/issue/{issueIdOrKey}/transitions.
type DoTransitionRequest struct {
	TransitionID    string           `json:"-"`
	Fields          map[string]any   `json:"fields,omitempty"`
	Update          map[string][]any `json:"update,omitempty"`
	Properties      []map[string]any `json:"properties,omitempty"`
	HistoryMetadata map[string]any   `json:"historyMetadata,omitempty"`
}
