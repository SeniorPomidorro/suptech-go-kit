package atlassian

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
)

// IssuesService provides Jira issue operations.
type IssuesService struct {
	client *Client
}

// FindIssuesOptions controls JQL search.
type FindIssuesOptions struct {
	Fields   []string
	Expand   []string
	PageSize int
	StartAt  int
	FetchAll bool
}

// CommentOption mutates comment creation payload.
type CommentOption func(payload map[string]any)

// WithCommentVisibility sets Jira comment visibility.
func WithCommentVisibility(visibilityType, value string) CommentOption {
	return func(payload map[string]any) {
		if strings.TrimSpace(visibilityType) == "" || strings.TrimSpace(value) == "" {
			return
		}
		payload["visibility"] = map[string]string{
			"type":  visibilityType,
			"value": value,
		}
	}
}

// WithCommentProperty appends arbitrary comment property.
func WithCommentProperty(key string, value any) CommentOption {
	return func(payload map[string]any) {
		if strings.TrimSpace(key) == "" {
			return
		}
		properties, _ := payload["properties"].([]map[string]any)
		properties = append(properties, map[string]any{
			"key":   key,
			"value": value,
		})
		payload["properties"] = properties
	}
}

// GetIssue returns Jira issue by key.
func (s *IssuesService) GetIssue(ctx context.Context, ticketKey string) (*Issue, error) {
	if strings.TrimSpace(ticketKey) == "" {
		return nil, errors.New("atlassian: ticket key is required")
	}

	path := fmt.Sprintf("/rest/api/3/issue/%s", url.PathEscape(ticketKey))
	req, err := s.client.newRequest(ctx, http.MethodGet, path, nil, nil)
	if err != nil {
		return nil, err
	}

	var issue Issue
	if err := s.client.transport.DoJSON(req, &issue); err != nil {
		return nil, err
	}
	return &issue, nil
}

// SaveStoryPoints updates Jira custom field for story points.
func (s *IssuesService) SaveStoryPoints(ctx context.Context, ticketKey string, points float64, fieldID string) error {
	if strings.TrimSpace(ticketKey) == "" {
		return errors.New("atlassian: ticket key is required")
	}
	if strings.TrimSpace(fieldID) == "" {
		return errors.New("atlassian: field ID is required")
	}

	path := fmt.Sprintf("/rest/api/3/issue/%s", url.PathEscape(ticketKey))
	payload := map[string]any{
		"fields": map[string]any{
			fieldID: points,
		},
	}

	req, err := s.client.newRequest(ctx, http.MethodPut, path, nil, payload)
	if err != nil {
		return err
	}
	return s.client.doNoResponseBody(req)
}

// FindIssues executes JQL search and optionally fetches all pages.
func (s *IssuesService) FindIssues(ctx context.Context, jql string, opts *FindIssuesOptions) (*SearchResult, error) {
	if strings.TrimSpace(jql) == "" {
		return nil, errors.New("atlassian: jql is required")
	}

	if opts == nil {
		opts = &FindIssuesOptions{}
	}

	pageSize := opts.PageSize
	if pageSize <= 0 {
		pageSize = 100
	}

	startAt := opts.StartAt
	result := &SearchResult{
		StartAt:    opts.StartAt,
		MaxResults: pageSize,
	}

	for {
		payload := map[string]any{
			"jql":        jql,
			"startAt":    startAt,
			"maxResults": pageSize,
		}
		if len(opts.Fields) > 0 {
			payload["fields"] = opts.Fields
		}
		if len(opts.Expand) > 0 {
			payload["expand"] = opts.Expand
		}

		req, err := s.client.newRequest(ctx, http.MethodPost, "/rest/api/3/search/jql", nil, payload)
		if err != nil {
			return nil, err
		}

		var page SearchResult
		if err := s.client.transport.DoJSON(req, &page); err != nil {
			return nil, err
		}

		if !opts.FetchAll {
			return &page, nil
		}

		result.Total = page.Total
		result.Issues = append(result.Issues, page.Issues...)
		startAt += len(page.Issues)
		if startAt >= page.Total || len(page.Issues) == 0 {
			return result, nil
		}
	}
}

// ManageTags updates Jira labels via add/remove or full replace.
func (s *IssuesService) ManageTags(ctx context.Context, ticketKey string, add, remove, replace []string) error {
	if strings.TrimSpace(ticketKey) == "" {
		return errors.New("atlassian: ticket key is required")
	}

	payload := map[string]any{}
	if len(replace) > 0 {
		payload["fields"] = map[string]any{"labels": replace}
	} else {
		ops := make([]map[string]string, 0, len(add)+len(remove))
		for _, label := range add {
			if strings.TrimSpace(label) == "" {
				continue
			}
			ops = append(ops, map[string]string{"add": label})
		}
		for _, label := range remove {
			if strings.TrimSpace(label) == "" {
				continue
			}
			ops = append(ops, map[string]string{"remove": label})
		}
		if len(ops) == 0 {
			return nil
		}
		payload["update"] = map[string]any{"labels": ops}
	}

	path := fmt.Sprintf("/rest/api/3/issue/%s", url.PathEscape(ticketKey))
	req, err := s.client.newRequest(ctx, http.MethodPut, path, nil, payload)
	if err != nil {
		return err
	}
	return s.client.doNoResponseBody(req)
}

// CreateComment creates Jira comment; internal=true adds JSM internal property.
func (s *IssuesService) CreateComment(ctx context.Context, ticketKey, text string, internal bool, opts ...CommentOption) (*Comment, error) {
	if strings.TrimSpace(ticketKey) == "" {
		return nil, errors.New("atlassian: ticket key is required")
	}
	if strings.TrimSpace(text) == "" {
		return nil, errors.New("atlassian: comment text is required")
	}

	payload := map[string]any{"body": text}
	if internal {
		payload["properties"] = []map[string]any{{
			"key":   "sd.public.comment",
			"value": map[string]any{"internal": true},
		}}
	}

	for _, opt := range opts {
		if opt != nil {
			opt(payload)
		}
	}

	path := fmt.Sprintf("/rest/api/3/issue/%s/comment", url.PathEscape(ticketKey))
	req, err := s.client.newRequest(ctx, http.MethodPost, path, nil, payload)
	if err != nil {
		return nil, err
	}

	var comment Comment
	if err := s.client.transport.DoJSON(req, &comment); err != nil {
		return nil, err
	}
	return &comment, nil
}

// AddAttachment uploads attachment to Jira issue.
func (s *IssuesService) AddAttachment(ctx context.Context, ticketKey, filename string, content []byte) (*Attachment, error) {
	if strings.TrimSpace(ticketKey) == "" {
		return nil, errors.New("atlassian: ticket key is required")
	}
	if strings.TrimSpace(filename) == "" {
		return nil, errors.New("atlassian: filename is required")
	}

	var payload bytes.Buffer
	writer := multipart.NewWriter(&payload)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return nil, fmt.Errorf("atlassian: create multipart file part: %w", err)
	}
	if _, err := part.Write(content); err != nil {
		return nil, fmt.Errorf("atlassian: write attachment content: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("atlassian: close multipart writer: %w", err)
	}

	path := fmt.Sprintf("/rest/api/3/issue/%s/attachments", url.PathEscape(ticketKey))
	req, err := s.client.newRawRequest(ctx, http.MethodPost, path, nil, payload.Bytes(), writer.FormDataContentType())
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Atlassian-Token", "no-check")

	var attachments []Attachment
	if err := s.client.transport.DoJSON(req, &attachments); err != nil {
		return nil, err
	}
	if len(attachments) == 0 {
		return &Attachment{}, nil
	}
	return &attachments[0], nil
}
