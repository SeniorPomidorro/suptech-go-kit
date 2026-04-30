package zoom

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// User is a subset of GET /users/{userId} response fields we use.
// Reference: docs/api-specs/ZoomUsers.json paths."/users/{userId}".get.
type User struct {
	ID            string `json:"id"`
	FirstName     string `json:"first_name"`
	LastName      string `json:"last_name"`
	DisplayName   string `json:"display_name"`
	Email         string `json:"email"`
	Type          int    `json:"type"`
	Status        string `json:"status"`
	AccountID     string `json:"account_id"`
	Timezone      string `json:"timezone"`
	PMI           int64  `json:"pmi"`
	UsePMI        bool   `json:"use_pmi"`
	PersonalURL   string `json:"personal_meeting_url"`
	Verified      int    `json:"verified"`
	RoleID        string `json:"role_id"`
	RoleName      string `json:"role_name"`
}

// UsersService groups user endpoints.
type UsersService struct {
	client *Client
}

// Get returns user info. The path parameter accepts either a userId or an email
// for endpoints that allow it (per Zoom convention for /users/{userId}).
//
// Endpoint: GET /users/{userId}.
// Required scope (granular): user:read:user:admin (we currently rely on
// user:read:list_users:admin and may need to switch to listing if Get returns 403).
func (s *UsersService) Get(ctx context.Context, userIDOrEmail string) (*User, error) {
	if strings.TrimSpace(userIDOrEmail) == "" {
		return nil, fmt.Errorf("zoom: userId is required")
	}
	path := fmt.Sprintf("/users/%s", url.PathEscape(userIDOrEmail))
	var out User
	if err := s.client.doJSON(ctx, http.MethodGet, path, nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
