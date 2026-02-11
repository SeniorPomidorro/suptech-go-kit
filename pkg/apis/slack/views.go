package slack

import (
	"context"
	"errors"
	"strings"
)

// ViewsService provides Slack modal/home views operations.
type ViewsService struct {
	client *Client
}

// OpenView opens a view for the provided trigger.
func (s *ViewsService) OpenView(ctx context.Context, triggerID string, req *ModalViewRequest) (*OpenViewResult, error) {
	if strings.TrimSpace(triggerID) == "" {
		return nil, errors.New("slack: trigger ID is required")
	}
	if req == nil {
		return nil, errors.New("slack: view request is required")
	}

	payload := map[string]any{
		"trigger_id": triggerID,
		"view":       req,
	}

	httpReq, err := s.client.newJSONRequest(ctx, "views.open", payload)
	if err != nil {
		return nil, err
	}

	var result OpenViewResult
	if err := s.client.do(httpReq, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// UpdateView updates an existing view by viewID or externalID.
func (s *ViewsService) UpdateView(ctx context.Context, view ModalViewRequest, externalID, hash, viewID string) (*OpenViewResult, error) {
	if strings.TrimSpace(view.Type) == "" {
		return nil, errors.New("slack: view type is required")
	}
	if strings.TrimSpace(viewID) == "" && strings.TrimSpace(externalID) == "" {
		return nil, errors.New("slack: view ID or external ID is required")
	}

	payload := map[string]any{
		"view": view,
	}
	if strings.TrimSpace(viewID) != "" {
		payload["view_id"] = viewID
	}
	if strings.TrimSpace(externalID) != "" {
		payload["external_id"] = externalID
	}
	if strings.TrimSpace(hash) != "" {
		payload["hash"] = hash
	}

	httpReq, err := s.client.newJSONRequest(ctx, "views.update", payload)
	if err != nil {
		return nil, err
	}

	var result OpenViewResult
	if err := s.client.do(httpReq, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
