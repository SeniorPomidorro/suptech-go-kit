package atlassian

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// UsersService provides Jira user lookups.
type UsersService struct {
	client *Client
}

// FindUsersOptions controls user search query.
type FindUsersOptions struct {
	StartAt         int
	MaxResults      int
	AccountID       string
	IncludeInactive bool
}

// FindUsers queries Jira users.
func (s *UsersService) FindUsers(ctx context.Context, query string, opts *FindUsersOptions) ([]User, error) {
	if strings.TrimSpace(query) == "" {
		return nil, errors.New("atlassian: query is required")
	}

	if opts == nil {
		opts = &FindUsersOptions{}
	}

	params := url.Values{}
	params.Set("query", query)
	if opts.StartAt > 0 {
		params.Set("startAt", strconv.Itoa(opts.StartAt))
	}
	if opts.MaxResults > 0 {
		params.Set("maxResults", strconv.Itoa(opts.MaxResults))
	}
	if opts.AccountID != "" {
		params.Set("accountId", opts.AccountID)
	}
	if opts.IncludeInactive {
		params.Set("includeInactive", "true")
	}

	req, err := s.client.newRequest(ctx, http.MethodGet, "/rest/api/3/user/search", params, nil)
	if err != nil {
		return nil, err
	}

	var users []User
	if err := s.client.transport.DoJSON(req, &users); err != nil {
		return nil, err
	}
	return users, nil
}
