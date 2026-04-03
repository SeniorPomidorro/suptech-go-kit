package slack

import (
	"context"
	"errors"
	"net/url"
	"strconv"
	"strings"
)

// ConversationsService provides Slack conversation operations.
type ConversationsService struct {
	client *Client
}

// GetConversationList returns conversations and follows cursor pagination.
func (s *ConversationsService) GetConversationList(ctx context.Context, excludeArchived bool, channelTypes []string) ([]Conversation, error) {
	var (
		cursor string
		all    []Conversation
	)

	for {
		params := url.Values{}
		if excludeArchived {
			params.Set("exclude_archived", "true")
		}
		if len(channelTypes) > 0 {
			params.Set("types", strings.Join(channelTypes, ","))
		}
		if cursor != "" {
			params.Set("cursor", cursor)
		}
		s.client.withTeamID(params)

		req, err := s.client.newGetRequest(ctx, "conversations.list", params)
		if err != nil {
			return nil, err
		}

		var response struct {
			Channels         []Conversation   `json:"channels"`
			ResponseMetadata ResponseMetadata `json:"response_metadata"`
		}
		if err := s.client.do(req, &response); err != nil {
			return nil, err
		}
		all = append(all, response.Channels...)

		cursor = strings.TrimSpace(response.ResponseMetadata.NextCursor)
		if cursor == "" {
			return all, nil
		}
	}
}

// CreateConversation creates a Slack channel.
func (s *ConversationsService) CreateConversation(ctx context.Context, name string, isPrivate bool) (*Conversation, error) {
	if strings.TrimSpace(name) == "" {
		return nil, errors.New("slack: conversation name is required")
	}

	form := url.Values{}
	form.Set("name", name)
	if isPrivate {
		form.Set("is_private", "true")
	}
	s.client.withTeamID(form)

	req, err := s.client.newFormRequest(ctx, "conversations.create", form)
	if err != nil {
		return nil, err
	}

	var response struct {
		Channel Conversation `json:"channel"`
	}
	if err := s.client.do(req, &response); err != nil {
		return nil, err
	}
	return &response.Channel, nil
}

// GetChannelByID returns conversation by channel ID.
func (s *ConversationsService) GetChannelByID(ctx context.Context, channelID string) (*Conversation, error) {
	if strings.TrimSpace(channelID) == "" {
		return nil, errors.New("slack: channel ID is required")
	}

	params := url.Values{}
	params.Set("channel", channelID)

	req, err := s.client.newGetRequest(ctx, "conversations.info", params)
	if err != nil {
		return nil, err
	}

	var response struct {
		Channel Conversation `json:"channel"`
	}
	if err := s.client.do(req, &response); err != nil {
		return nil, err
	}
	return &response.Channel, nil
}

// GetHistory fetches message history from a conversation.
// Uses conversations.history Slack API method.
func (s *ConversationsService) GetHistory(ctx context.Context, req *GetHistoryRequest) (*HistoryResponse, error) {
	if req == nil {
		return nil, errors.New("slack: request is required")
	}
	if strings.TrimSpace(req.Channel) == "" {
		return nil, errors.New("slack: channel ID is required")
	}

	params := url.Values{}
	params.Set("channel", req.Channel)
	if req.Cursor != "" {
		params.Set("cursor", req.Cursor)
	}
	if req.IncludeAllMetadata {
		params.Set("include_all_metadata", "true")
	}
	if req.Inclusive {
		params.Set("inclusive", "true")
	}
	if req.Latest != "" {
		params.Set("latest", req.Latest)
	}
	if req.Oldest != "" {
		params.Set("oldest", req.Oldest)
	}
	if req.Limit > 0 {
		params.Set("limit", strconv.Itoa(req.Limit))
	}

	httpReq, err := s.client.newGetRequest(ctx, "conversations.history", params)
	if err != nil {
		return nil, err
	}

	var response HistoryResponse
	if err := s.client.do(httpReq, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

// GetReplies fetches replies (thread messages) for a given parent message.
// Uses conversations.replies Slack API method.
func (s *ConversationsService) GetReplies(ctx context.Context, req *GetRepliesRequest) (*HistoryResponse, error) {
	if req == nil {
		return nil, errors.New("slack: request is required")
	}
	if strings.TrimSpace(req.Channel) == "" {
		return nil, errors.New("slack: channel ID is required")
	}
	if strings.TrimSpace(req.TS) == "" {
		return nil, errors.New("slack: thread timestamp is required")
	}

	params := url.Values{}
	params.Set("channel", req.Channel)
	params.Set("ts", req.TS)
	if req.Cursor != "" {
		params.Set("cursor", req.Cursor)
	}
	if req.IncludeAllMetadata {
		params.Set("include_all_metadata", "true")
	}
	if req.Inclusive {
		params.Set("inclusive", "true")
	}
	if req.Latest != "" {
		params.Set("latest", req.Latest)
	}
	if req.Oldest != "" {
		params.Set("oldest", req.Oldest)
	}
	if req.Limit > 0 {
		params.Set("limit", strconv.Itoa(req.Limit))
	}

	httpReq, err := s.client.newGetRequest(ctx, "conversations.replies", params)
	if err != nil {
		return nil, err
	}

	var response HistoryResponse
	if err := s.client.do(httpReq, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

// InviteUsersToChannel invites users to a channel.
func (s *ConversationsService) InviteUsersToChannel(ctx context.Context, userIDs []string, channelID string) (*Conversation, error) {
	if strings.TrimSpace(channelID) == "" {
		return nil, errors.New("slack: channel ID is required")
	}
	if len(userIDs) == 0 {
		return nil, errors.New("slack: at least one user ID is required")
	}

	form := url.Values{}
	form.Set("channel", channelID)
	form.Set("users", strings.Join(userIDs, ","))
	s.client.withTeamID(form)

	req, err := s.client.newFormRequest(ctx, "conversations.invite", form)
	if err != nil {
		return nil, err
	}

	var response struct {
		Channel Conversation `json:"channel"`
	}
	if err := s.client.do(req, &response); err != nil {
		return nil, err
	}
	return &response.Channel, nil
}
