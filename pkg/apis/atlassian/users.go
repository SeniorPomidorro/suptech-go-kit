package atlassian

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const defaultBulkGetUsersFetchAllThrottle = 200 * time.Millisecond

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

// BulkGetUsers fetches Jira users by account IDs with optional pagination.
func (s *UsersService) BulkGetUsers(ctx context.Context, opts *BulkGetUsersOptions) (*BulkUsersResult, error) {
	if opts == nil {
		return nil, errors.New("atlassian: at least one account ID is required")
	}

	usernames := make([]string, 0, len(opts.Usernames))
	for _, username := range opts.Usernames {
		if trimmed := strings.TrimSpace(username); trimmed != "" {
			usernames = append(usernames, trimmed)
		}
	}
	keys := make([]string, 0, len(opts.Keys))
	for _, key := range opts.Keys {
		if trimmed := strings.TrimSpace(key); trimmed != "" {
			keys = append(keys, trimmed)
		}
	}
	accountIDs := make([]string, 0, len(opts.AccountIDs))
	for _, accountID := range opts.AccountIDs {
		if trimmed := strings.TrimSpace(accountID); trimmed != "" {
			accountIDs = append(accountIDs, trimmed)
		}
	}
	if len(accountIDs) == 0 {
		return nil, errors.New("atlassian: at least one account ID is required")
	}

	startAt := opts.StartAt
	combined := make([]User, 0)
	throttle := opts.FetchAllThrottle
	if throttle <= 0 {
		throttle = defaultBulkGetUsersFetchAllThrottle
	}

	for {
		params := url.Values{}
		if startAt > 0 {
			params.Set("startAt", strconv.Itoa(startAt))
		}
		if opts.MaxResults > 0 {
			params.Set("maxResults", strconv.Itoa(opts.MaxResults))
		}
		for _, username := range usernames {
			params.Add("username", username)
		}
		for _, key := range keys {
			params.Add("key", key)
		}
		for _, accountID := range accountIDs {
			params.Add("accountId", accountID)
		}

		req, err := s.client.newRequest(ctx, http.MethodGet, "/rest/api/3/user/bulk", params, nil)
		if err != nil {
			return nil, err
		}

		var page BulkUsersResult
		if err := s.client.transport.DoJSON(req, &page); err != nil {
			return nil, err
		}
		if !opts.FetchAll {
			return &page, nil
		}

		combined = append(combined, page.Values...)
		if page.IsLast || page.NextPage == "" || len(page.Values) == 0 {
			page.Values = combined
			page.StartAt = opts.StartAt
			page.NextPage = ""
			page.IsLast = true
			return &page, nil
		}
		if err := sleepWithContext(ctx, throttle); err != nil {
			return nil, err
		}
		startAt = page.StartAt + len(page.Values)
	}
}

func sleepWithContext(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
