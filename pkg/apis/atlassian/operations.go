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
func (s *OperationsService) CreateAlert(ctx context.Context, payload map[string]any) (*Alert, error) {
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

	var alert Alert
	if err := s.client.transport.DoJSON(req, &alert); err != nil {
		return nil, err
	}
	return &alert, nil
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
func (s *OperationsService) ListAlerts(ctx context.Context, opts ListAlertsOptions) (*AlertsListResult, error) {
	path, err := s.client.opsPath("/alerts")
	if err != nil {
		return nil, err
	}

	query := url.Values{}
	if strings.TrimSpace(opts.Query) != "" {
		query.Set("query", opts.Query)
	}
	if opts.Limit > 0 {
		query.Set("limit", strconv.Itoa(opts.Limit))
	}
	if opts.Offset > 0 {
		query.Set("offset", strconv.Itoa(opts.Offset))
	}
	if strings.TrimSpace(opts.Order) != "" {
		query.Set("order", opts.Order)
	}

	req, err := s.client.newCloudRequest(ctx, http.MethodGet, path, query, nil)
	if err != nil {
		return nil, err
	}

	var result AlertsListResult
	if err := s.client.transport.DoJSON(req, &result); err != nil {
		return nil, err
	}
	if len(result.Values) == 0 {
		result.Values = result.items()
	}
	return &result, nil
}

// EnableOpsForTeam enables Ops capabilities for a team.
func (s *OperationsService) EnableOpsForTeam(ctx context.Context, teamID string) error {
	if strings.TrimSpace(teamID) == "" {
		return errors.New("atlassian: team ID is required")
	}

	path, err := s.client.opsPath("/teams/" + url.PathEscape(teamID) + "/enable")
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
func (s *OperationsService) ListTeams(ctx context.Context, opts ListTeamsOptions) (*TeamsListResult, error) {
	path, err := s.client.opsPath("/teams")
	if err != nil {
		return nil, err
	}

	query := url.Values{}
	if strings.TrimSpace(opts.Query) != "" {
		query.Set("query", opts.Query)
	}
	if opts.Limit > 0 {
		query.Set("limit", strconv.Itoa(opts.Limit))
	}
	if opts.Offset > 0 {
		query.Set("offset", strconv.Itoa(opts.Offset))
	}

	req, err := s.client.newCloudRequest(ctx, http.MethodGet, path, query, nil)
	if err != nil {
		return nil, err
	}

	var result TeamsListResult
	if err := s.client.transport.DoJSON(req, &result); err != nil {
		return nil, err
	}
	if len(result.Values) == 0 {
		result.Values = result.items()
	}
	return &result, nil
}

// GetUserNotificationSettings returns operations notification settings for user.
func (s *OperationsService) GetUserNotificationSettings(ctx context.Context, userID string, opts GetUserNotificationSettingsOptions) (*UserNotificationSettings, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, errors.New("atlassian: user ID is required")
	}

	path, err := s.client.opsPath("/users/" + url.PathEscape(userID) + "/notification-settings")
	if err != nil {
		return nil, err
	}

	query := url.Values{}
	if strings.TrimSpace(opts.TeamID) != "" {
		query.Set("teamId", opts.TeamID)
	}

	req, err := s.client.newCloudRequest(ctx, http.MethodGet, path, query, nil)
	if err != nil {
		return nil, err
	}

	var settings UserNotificationSettings
	if err := s.client.transport.DoJSON(req, &settings); err != nil {
		return nil, err
	}
	return &settings, nil
}

// ListSchedules lists schedules.
func (s *OperationsService) ListSchedules(ctx context.Context, opts ListSchedulesOptions) (*SchedulesListResult, error) {
	path, err := s.client.opsPath("/schedules")
	if err != nil {
		return nil, err
	}

	query := url.Values{}
	if strings.TrimSpace(opts.TeamID) != "" {
		query.Set("teamId", opts.TeamID)
	}
	if opts.Limit > 0 {
		query.Set("limit", strconv.Itoa(opts.Limit))
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
	if len(result.Values) == 0 {
		result.Values = result.items()
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

// ListOnCallResponders lists on-call responders for schedule.
func (s *OperationsService) ListOnCallResponders(ctx context.Context, scheduleID string, opts ListOnCallRespondersOptions) ([]OnCallResponder, error) {
	if strings.TrimSpace(scheduleID) == "" {
		return nil, errors.New("atlassian: schedule ID is required")
	}

	path, err := s.client.opsPath("/schedules/" + url.PathEscape(scheduleID) + "/on-call/responders")
	if err != nil {
		return nil, err
	}

	query := url.Values{}
	if strings.TrimSpace(opts.From) != "" {
		query.Set("from", opts.From)
	}
	if strings.TrimSpace(opts.To) != "" {
		query.Set("to", opts.To)
	}
	if opts.Limit > 0 {
		query.Set("limit", strconv.Itoa(opts.Limit))
	}

	req, err := s.client.newCloudRequest(ctx, http.MethodGet, path, query, nil)
	if err != nil {
		return nil, err
	}

	var result OnCallRespondersResult
	if err := s.client.transport.DoJSON(req, &result); err != nil {
		return nil, err
	}
	return result.items(), nil
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
