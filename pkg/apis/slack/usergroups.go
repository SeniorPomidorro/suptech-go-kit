package slack

import (
	"context"
	"errors"
	"net/url"
	"strings"
)

// UserGroupsService provides Slack user groups operations.
type UserGroupsService struct {
	client *Client
}

// CreateUserGroup creates a user group in Slack.
func (s *UserGroupsService) CreateUserGroup(ctx context.Context, name, tag string) (*UserGroup, error) {
	if strings.TrimSpace(name) == "" {
		return nil, errors.New("slack: user group name is required")
	}
	if strings.TrimSpace(tag) == "" {
		return nil, errors.New("slack: user group tag is required")
	}

	form := url.Values{}
	form.Set("name", name)
	form.Set("handle", tag)
	s.client.withTeamID(form)

	req, err := s.client.newFormRequest(ctx, "usergroups.create", form)
	if err != nil {
		return nil, err
	}

	var response struct {
		UserGroup UserGroup `json:"usergroup"`
	}
	if err := s.client.do(req, &response); err != nil {
		return nil, err
	}
	return &response.UserGroup, nil
}

// ListUserGroups lists user groups.
func (s *UserGroupsService) ListUserGroups(ctx context.Context) ([]UserGroup, error) {
	form := url.Values{}
	s.client.withTeamID(form)

	req, err := s.client.newFormRequest(ctx, "usergroups.list", form)
	if err != nil {
		return nil, err
	}

	var response struct {
		UserGroups []UserGroup `json:"usergroups"`
	}
	if err := s.client.do(req, &response); err != nil {
		return nil, err
	}
	return response.UserGroups, nil
}
