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
	params := url.Values{}
	s.client.withTeamID(params)

	req, err := s.client.newGetRequest(ctx, "usergroups.list", params)
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

// ListUserGroupUsers lists members of a user group using usergroups.users.list.
func (s *UserGroupsService) ListUserGroupUsers(ctx context.Context, req *ListUserGroupUsersRequest) ([]string, error) {
	if req == nil {
		return nil, errors.New("slack: request is required")
	}
	userGroupID := strings.TrimSpace(req.UserGroup)
	if userGroupID == "" {
		return nil, errors.New("slack: user group ID is required")
	}
	teamID := strings.TrimSpace(req.TeamID)

	params := url.Values{}
	params.Set("usergroup", userGroupID)
	if req.IncludeDisabled {
		params.Set("include_disabled", "true")
	}
	if teamID != "" {
		params.Set("team_id", teamID)
	} else {
		s.client.withTeamID(params)
	}

	httpReq, err := s.client.newGetRequest(ctx, "usergroups.users.list", params)
	if err != nil {
		return nil, err
	}

	var response struct {
		Users []string `json:"users"`
	}
	if err := s.client.do(httpReq, &response); err != nil {
		return nil, err
	}
	if len(response.Users) == 0 {
		return []string{}, nil
	}
	return response.Users, nil
}

// UpdateUserGroupUsers updates members of a user group using usergroups.users.update.
func (s *UserGroupsService) UpdateUserGroupUsers(ctx context.Context, req *UpdateUserGroupUsersRequest) (*UserGroup, error) {
	if req == nil {
		return nil, errors.New("slack: request is required")
	}
	userGroupID := strings.TrimSpace(req.UserGroup)
	if userGroupID == "" {
		return nil, errors.New("slack: user group ID is required")
	}
	teamID := strings.TrimSpace(req.TeamID)

	users := make([]string, 0, len(req.Users))
	for _, user := range req.Users {
		if trimmed := strings.TrimSpace(user); trimmed != "" {
			users = append(users, trimmed)
		}
	}
	if len(users) == 0 {
		return nil, errors.New("slack: at least one user ID is required")
	}

	form := url.Values{}
	form.Set("usergroup", userGroupID)
	form.Set("users", strings.Join(users, ","))
	if req.IncludeCount {
		form.Set("include_count", "true")
	}
	if req.IsShared {
		form.Set("is_shared", "true")
	}

	channels := make([]string, 0, len(req.AdditionalChannels))
	for _, channel := range req.AdditionalChannels {
		if trimmed := strings.TrimSpace(channel); trimmed != "" {
			channels = append(channels, trimmed)
		}
	}
	if len(channels) > 0 {
		form.Set("additional_channels", strings.Join(channels, ","))
	}

	if teamID != "" {
		form.Set("team_id", teamID)
	} else {
		s.client.withTeamID(form)
	}

	httpReq, err := s.client.newFormRequest(ctx, "usergroups.users.update", form)
	if err != nil {
		return nil, err
	}

	var response struct {
		UserGroup UserGroup `json:"usergroup"`
	}
	if err := s.client.do(httpReq, &response); err != nil {
		return nil, err
	}
	return &response.UserGroup, nil
}
