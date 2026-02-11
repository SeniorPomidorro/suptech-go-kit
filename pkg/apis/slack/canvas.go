package slack

import (
	"context"
	"errors"
	"strings"
)

// CanvasService provides Slack canvas operations.
type CanvasService struct {
	client *Client
}

// CreateCanvas creates a canvas with initial markdown content.
func (s *CanvasService) CreateCanvas(ctx context.Context, title, text string) (*Canvas, error) {
	if strings.TrimSpace(title) == "" {
		return nil, errors.New("slack: canvas title is required")
	}

	payload := map[string]any{
		"title": title,
		"document_content": map[string]any{
			"type":     "markdown",
			"markdown": text,
		},
	}
	httpReq, err := s.client.newJSONRequest(ctx, "canvases.create", payload)
	if err != nil {
		return nil, err
	}

	var response struct {
		Canvas Canvas `json:"canvas"`
	}
	if err := s.client.do(httpReq, &response); err != nil {
		return nil, err
	}
	return &response.Canvas, nil
}

// ShareCanvas shares canvas access with a channel.
func (s *CanvasService) ShareCanvas(ctx context.Context, canvasID, channelID string) error {
	if strings.TrimSpace(canvasID) == "" {
		return errors.New("slack: canvas ID is required")
	}
	if strings.TrimSpace(channelID) == "" {
		return errors.New("slack: channel ID is required")
	}

	payload := map[string]any{
		"canvas_id":   canvasID,
		"channel_ids": []string{channelID},
	}
	httpReq, err := s.client.newJSONRequest(ctx, "canvases.access.set", payload)
	if err != nil {
		return err
	}
	return s.client.do(httpReq, nil)
}
