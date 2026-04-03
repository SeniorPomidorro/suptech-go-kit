package atlassian

import "encoding/json"

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
