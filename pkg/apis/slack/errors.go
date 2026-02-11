package slack

import "fmt"

// Error describes Slack API errors when JSON contains ok=false.
type Error struct {
	Code     string
	Needed   string
	Provided string
	Warning  string
}

// Error formats Slack API error details.
func (e *Error) Error() string {
	if e == nil {
		return "slack: api error"
	}
	if e.Code == "" {
		return "slack: api error"
	}
	return fmt.Sprintf("slack: api error code=%s", e.Code)
}
