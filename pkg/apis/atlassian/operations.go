package atlassian

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// OperationsService provides Jira Operations API methods.
type OperationsService struct {
	client *Client
}

// CreateAlert creates a new operations alert.
func (s *OperationsService) CreateAlert(ctx context.Context, payload map[string]any) (*CreateAlertResponse, error) {
	if len(payload) == 0 {
		return nil, errors.New("atlassian: alert payload is required")
	}

	path, err := s.client.opsPath("/alerts")
	if err != nil {
		return nil, err
	}

	req, err := s.client.newCloudRequest(ctx, http.MethodPost, path, nil, payload)
	if err != nil {
		return nil, err
	}

	var resp CreateAlertResponse
	if err := s.client.transport.DoJSON(req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// GetAlert returns alert by ID.
func (s *OperationsService) GetAlert(ctx context.Context, alertID string) (*Alert, error) {
	if strings.TrimSpace(alertID) == "" {
		return nil, errors.New("atlassian: alert ID is required")
	}

	path, err := s.client.opsPath("/alerts/" + url.PathEscape(alertID))
	if err != nil {
		return nil, err
	}

	req, err := s.client.newCloudRequest(ctx, http.MethodGet, path, nil, nil)
	if err != nil {
		return nil, err
	}

	var alert Alert
	if err := s.client.transport.DoJSON(req, &alert); err != nil {
		return nil, err
	}
	return &alert, nil
}

// ListAlerts lists alerts with optional filters.
func (s *OperationsService) ListAlerts(ctx context.Context, opts *ListAlertsOptions) (*AlertsListResult, error) {
	path, err := s.client.opsPath("/alerts")
	if err != nil {
		return nil, err
	}

	if opts == nil {
		opts = &ListAlertsOptions{}
	}

	query := url.Values{}
	if strings.TrimSpace(opts.Query) != "" {
		query.Set("query", opts.Query)
	}
	if opts.Size > 0 {
		query.Set("size", strconv.Itoa(opts.Size))
	}
	if opts.Offset > 0 {
		query.Set("offset", strconv.Itoa(opts.Offset))
	}
	if strings.TrimSpace(opts.Order) != "" {
		query.Set("order", opts.Order)
	}
	if strings.TrimSpace(opts.Sort) != "" {
		query.Set("sort", opts.Sort)
	}

	req, err := s.client.newCloudRequest(ctx, http.MethodGet, path, query, nil)
	if err != nil {
		return nil, err
	}

	var result AlertsListResult
	if err := s.client.transport.DoJSON(req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// EnableOpsForTeam enables Ops capabilities for a team.
func (s *OperationsService) EnableOpsForTeam(ctx context.Context, teamID string) error {
	if strings.TrimSpace(teamID) == "" {
		return errors.New("atlassian: team ID is required")
	}

	path, err := s.client.opsPath("/teams/" + url.PathEscape(teamID) + "/enable-ops")
	if err != nil {
		return err
	}

	req, err := s.client.newCloudRequest(ctx, http.MethodPost, path, nil, nil)
	if err != nil {
		return err
	}
	return s.client.doNoResponseBody(req)
}

// ListTeams lists teams in Jira Operations.
func (s *OperationsService) ListTeams(ctx context.Context) (*TeamsListResult, error) {
	path, err := s.client.opsPath("/teams")
	if err != nil {
		return nil, err
	}

	req, err := s.client.newCloudRequest(ctx, http.MethodGet, path, nil, nil)
	if err != nil {
		return nil, err
	}

	var result TeamsListResult
	if err := s.client.transport.DoJSON(req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ListNotificationRules lists operations notification rules.
func (s *OperationsService) ListNotificationRules(ctx context.Context, opts *ListNotificationRulesOptions) (*NotificationRulesResult, error) {
	path, err := s.client.opsPath("/notification-rules")
	if err != nil {
		return nil, err
	}

	if opts == nil {
		opts = &ListNotificationRulesOptions{}
	}

	query := url.Values{}
	if opts.Size > 0 {
		query.Set("size", strconv.Itoa(opts.Size))
	}
	if opts.Offset > 0 {
		query.Set("offset", strconv.Itoa(opts.Offset))
	}

	req, err := s.client.newCloudRequest(ctx, http.MethodGet, path, query, nil)
	if err != nil {
		return nil, err
	}

	var result NotificationRulesResult
	if err := s.client.transport.DoJSON(req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ListSchedules lists schedules.
func (s *OperationsService) ListSchedules(ctx context.Context, opts *ListSchedulesOptions) (*SchedulesListResult, error) {
	path, err := s.client.opsPath("/schedules")
	if err != nil {
		return nil, err
	}

	if opts == nil {
		opts = &ListSchedulesOptions{}
	}

	query := url.Values{}
	if strings.TrimSpace(opts.Query) != "" {
		query.Set("query", opts.Query)
	}
	if opts.Size > 0 {
		query.Set("size", strconv.Itoa(opts.Size))
	}
	if opts.Offset > 0 {
		query.Set("offset", strconv.Itoa(opts.Offset))
	}

	req, err := s.client.newCloudRequest(ctx, http.MethodGet, path, query, nil)
	if err != nil {
		return nil, err
	}

	var result SchedulesListResult
	if err := s.client.transport.DoJSON(req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetSchedule gets schedule by ID.
func (s *OperationsService) GetSchedule(ctx context.Context, scheduleID string) (*Schedule, error) {
	if strings.TrimSpace(scheduleID) == "" {
		return nil, errors.New("atlassian: schedule ID is required")
	}

	path, err := s.client.opsPath("/schedules/" + url.PathEscape(scheduleID))
	if err != nil {
		return nil, err
	}

	req, err := s.client.newCloudRequest(ctx, http.MethodGet, path, nil, nil)
	if err != nil {
		return nil, err
	}

	var schedule Schedule
	if err := s.client.transport.DoJSON(req, &schedule); err != nil {
		return nil, err
	}
	return &schedule, nil
}

// ListOnCalls returns on-call participants for a schedule.
func (s *OperationsService) ListOnCalls(ctx context.Context, scheduleID string, opts *ListOnCallOptions) (*OnCallResult, error) {
	if strings.TrimSpace(scheduleID) == "" {
		return nil, errors.New("atlassian: schedule ID is required")
	}

	path, err := s.client.opsPath("/schedules/" + url.PathEscape(scheduleID) + "/on-calls")
	if err != nil {
		return nil, err
	}

	if opts == nil {
		opts = &ListOnCallOptions{}
	}

	query := url.Values{}
	if opts.Flat {
		query.Set("flat", "true")
	}
	if strings.TrimSpace(opts.Date) != "" {
		query.Set("date", opts.Date)
	}

	req, err := s.client.newCloudRequest(ctx, http.MethodGet, path, query, nil)
	if err != nil {
		return nil, err
	}

	var result OnCallResult
	if err := s.client.transport.DoJSON(req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) opsPath(pathSuffix string) (string, error) {
	cloudID := strings.TrimSpace(c.opsCloudID)
	if cloudID == "" {
		cloudID = strings.TrimSpace(c.assetsCloudID)
	}
	if cloudID == "" {
		return "", errors.New("atlassian: operations cloud ID is required")
	}

	base := "/jsm/ops/api/" + url.PathEscape(cloudID) + "/v1"
	if pathSuffix == "" {
		return base, nil
	}
	if strings.HasPrefix(pathSuffix, "/") {
		return base + pathSuffix, nil
	}
	return base + "/" + pathSuffix, nil
}
