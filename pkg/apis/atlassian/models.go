package atlassian

import "encoding/json"

// Issue is a minimal Jira issue DTO for client consumers.
type Issue struct {
	ID     string          `json:"id"`
	Key    string          `json:"key"`
	Fields json.RawMessage `json:"fields,omitempty"`
}

// SearchResult is Jira search response.
type SearchResult struct {
	StartAt    int     `json:"startAt"`
	MaxResults int     `json:"maxResults"`
	Total      int     `json:"total"`
	Issues     []Issue `json:"issues"`
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
