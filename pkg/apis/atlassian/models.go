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
