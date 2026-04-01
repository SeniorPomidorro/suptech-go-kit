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
func (s *MessagesService) PostMessage(ctx context.Context, req *PostMessageRequest) (*PostedMessage, error) {
	if req == nil {
		return nil, errors.New("slack: request is required")
	}
	if strings.TrimSpace(req.Channel) == "" {
		return nil, errors.New("slack: channel ID is required")
	}
	if strings.TrimSpace(req.Text) == "" && len(req.Blocks) == 0 && len(req.Attachments) == 0 {
		return nil, errors.New("slack: text, blocks, or attachments is required")
	}

	httpReq, err := s.client.newJSONRequest(ctx, "chat.postMessage", req)
	if err != nil {
		return nil, err
	}

	var response PostedMessage
	if err := s.client.do(httpReq, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

// PostEphemeralMessage posts an ephemeral message visible only to a specific user.
func (s *MessagesService) PostEphemeralMessage(ctx context.Context, req *PostEphemeralRequest) (*EphemeralPostResult, error) {
	if req == nil {
		return nil, errors.New("slack: request is required")
	}
	if strings.TrimSpace(req.Channel) == "" {
		return nil, errors.New("slack: channel ID is required")
	}
	if strings.TrimSpace(req.User) == "" {
		return nil, errors.New("slack: user ID is required")
	}
	if strings.TrimSpace(req.Text) == "" && len(req.Blocks) == 0 && len(req.Attachments) == 0 {
		return nil, errors.New("slack: text, blocks, or attachments is required")
	}

	httpReq, err := s.client.newJSONRequest(ctx, "chat.postEphemeral", req)
	if err != nil {
		return nil, err
	}

	var response EphemeralPostResult
	if err := s.client.do(httpReq, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

// UpdateMessage updates an existing message.
func (s *MessagesService) UpdateMessage(ctx context.Context, req *UpdateMessageRequest) (*PostedMessage, error) {
	if req == nil {
		return nil, errors.New("slack: request is required")
	}
	if strings.TrimSpace(req.Channel) == "" {
		return nil, errors.New("slack: channel ID is required")
	}
	if strings.TrimSpace(req.TS) == "" {
		return nil, errors.New("slack: ts is required")
	}
	if strings.TrimSpace(req.Text) == "" && len(req.Blocks) == 0 && len(req.Attachments) == 0 {
		return nil, errors.New("slack: text, blocks, or attachments is required")
	}

	httpReq, err := s.client.newJSONRequest(ctx, "chat.update", req)
	if err != nil {
		return nil, err
	}

	var response PostedMessage
	if err := s.client.do(httpReq, &response); err != nil {
		return nil, err
	}
	return &response, nil
}
