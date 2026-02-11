package slack

import (
	"context"
	"errors"
	"strings"
)

// MessagesService provides Slack messaging operations.
type MessagesService struct {
	client *Client
}

// PostMessage posts a message to a channel.
func (s *MessagesService) PostMessage(ctx context.Context, channelID, message string) (*PostedMessage, error) {
	if strings.TrimSpace(channelID) == "" {
		return nil, errors.New("slack: channel ID is required")
	}
	if strings.TrimSpace(message) == "" {
		return nil, errors.New("slack: message is required")
	}

	payload := map[string]any{
		"channel": channelID,
		"text":    message,
	}
	req, err := s.client.newJSONRequest(ctx, "chat.postMessage", payload)
	if err != nil {
		return nil, err
	}

	var response PostedMessage
	if err := s.client.do(req, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

// PostEphemeralMessage posts an ephemeral message.
func (s *MessagesService) PostEphemeralMessage(ctx context.Context, channelID, userID, message string) (*EphemeralPostResult, error) {
	if strings.TrimSpace(channelID) == "" {
		return nil, errors.New("slack: channel ID is required")
	}
	if strings.TrimSpace(userID) == "" {
		return nil, errors.New("slack: user ID is required")
	}
	if strings.TrimSpace(message) == "" {
		return nil, errors.New("slack: message is required")
	}

	payload := map[string]any{
		"channel": channelID,
		"user":    userID,
		"text":    message,
	}
	req, err := s.client.newJSONRequest(ctx, "chat.postEphemeral", payload)
	if err != nil {
		return nil, err
	}

	var response EphemeralPostResult
	if err := s.client.do(req, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

// UpdateMessage updates an existing message.
func (s *MessagesService) UpdateMessage(ctx context.Context, channelID, ts, message string) (*PostedMessage, error) {
	if strings.TrimSpace(channelID) == "" {
		return nil, errors.New("slack: channel ID is required")
	}
	if strings.TrimSpace(ts) == "" {
		return nil, errors.New("slack: ts is required")
	}
	if strings.TrimSpace(message) == "" {
		return nil, errors.New("slack: message is required")
	}

	payload := map[string]any{
		"channel": channelID,
		"ts":      ts,
		"text":    message,
	}
	req, err := s.client.newJSONRequest(ctx, "chat.update", payload)
	if err != nil {
		return nil, err
	}

	var response PostedMessage
	if err := s.client.do(req, &response); err != nil {
		return nil, err
	}
	return &response, nil
}
