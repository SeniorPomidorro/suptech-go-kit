package slack

import (
	"context"
	"errors"
	"net/url"
	"strings"
)

// UsersService provides Slack users operations.
type UsersService struct {
	client *Client
}

// GetUserByID returns user by ID.
func (s *UsersService) GetUserByID(ctx context.Context, userID string) (*User, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, errors.New("slack: user ID is required")
	}

	params := url.Values{}
	params.Set("user", userID)

	req, err := s.client.newGetRequest(ctx, "users.info", params)
	if err != nil {
		return nil, err
	}

	var response struct {
		User User `json:"user"`
	}
	if err := s.client.do(req, &response); err != nil {
		return nil, err
	}
	return &response.User, nil
}

// GetUsersByID returns users for provided IDs.
func (s *UsersService) GetUsersByID(ctx context.Context, userIDs []string) ([]User, error) {
	if len(userIDs) == 0 {
		return nil, errors.New("slack: at least one user ID is required")
	}

	users := make([]User, 0, len(userIDs))
	for _, id := range userIDs {
		user, err := s.GetUserByID(ctx, id)
		if err != nil {
			return nil, err
		}
		users = append(users, *user)
	}
	return users, nil
}

// GetUsersByGroupID returns users belonging to a user group.
func (s *UsersService) GetUsersByGroupID(ctx context.Context, userGroupID string) ([]User, error) {
	if strings.TrimSpace(userGroupID) == "" {
		return nil, errors.New("slack: user group ID is required")
	}

	params := url.Values{}
	params.Set("usergroup", userGroupID)
	s.client.withTeamID(params)

	req, err := s.client.newGetRequest(ctx, "usergroups.users.list", params)
	if err != nil {
		return nil, err
	}

	var response struct {
		Users []string `json:"users"`
	}
	if err := s.client.do(req, &response); err != nil {
		return nil, err
	}
	if len(response.Users) == 0 {
		return []User{}, nil
	}
	return s.GetUsersByID(ctx, response.Users)
}

// GetUserByEmail returns user by email.
func (s *UsersService) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	if strings.TrimSpace(email) == "" {
		return nil, errors.New("slack: email is required")
	}

	params := url.Values{}
	params.Set("email", email)

	req, err := s.client.newGetRequest(ctx, "users.lookupByEmail", params)
	if err != nil {
		return nil, err
	}

	var response struct {
		User User `json:"user"`
	}
	if err := s.client.do(req, &response); err != nil {
		return nil, err
	}
	return &response.User, nil
}
