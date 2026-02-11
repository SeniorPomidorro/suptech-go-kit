package slack

import (
	"context"
	"errors"
	"net/url"
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
		form := url.Values{}
		if excludeArchived {
			form.Set("exclude_archived", "true")
		}
		if len(channelTypes) > 0 {
			form.Set("types", strings.Join(channelTypes, ","))
		}
		if cursor != "" {
			form.Set("cursor", cursor)
		}
		s.client.withTeamID(form)

		req, err := s.client.newFormRequest(ctx, "conversations.list", form)
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

	form := url.Values{}
	form.Set("channel", channelID)
	s.client.withTeamID(form)

	req, err := s.client.newFormRequest(ctx, "conversations.info", form)
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
