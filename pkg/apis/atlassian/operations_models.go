package atlassian

// Alert is a minimal Jira Operations alert DTO.
type Alert struct {
	ID          string         `json:"id,omitempty"`
	Alias       string         `json:"alias,omitempty"`
	Message     string         `json:"message,omitempty"`
	Description string         `json:"description,omitempty"`
	Priority    string         `json:"priority,omitempty"`
	Status      string         `json:"status,omitempty"`
	Source      map[string]any `json:"source,omitempty"`
}

// AlertsListResult represents a paginated alerts response.
type AlertsListResult struct {
	Total  int     `json:"total"`
	Values []Alert `json:"values,omitempty"`
	Alerts []Alert `json:"alerts,omitempty"`
}

// ListAlertsOptions controls alert listing.
type ListAlertsOptions struct {
	Query  string
	Limit  int
	Offset int
	Order  string
}

// Team is a minimal Jira Operations team DTO.
type Team struct {
	ID      string `json:"id,omitempty"`
	Name    string `json:"name,omitempty"`
	Enabled bool   `json:"enabled,omitempty"`
}

// TeamsListResult represents a paginated team list.
type TeamsListResult struct {
	Total  int    `json:"total"`
	Values []Team `json:"values,omitempty"`
	Teams  []Team `json:"teams,omitempty"`
}

// ListTeamsOptions controls teams listing.
type ListTeamsOptions struct {
	Query  string
	Limit  int
	Offset int
}

// UserNotificationSettings contains user notification settings payload.
type UserNotificationSettings struct {
	UserID   string         `json:"userId,omitempty"`
	Channels []string       `json:"channels,omitempty"`
	Settings map[string]any `json:"settings,omitempty"`
}

// GetUserNotificationSettingsOptions controls optional query filters.
type GetUserNotificationSettingsOptions struct {
	TeamID string
}

// Schedule is a minimal operations schedule DTO.
type Schedule struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
}

// SchedulesListResult represents paginated schedules.
type SchedulesListResult struct {
	Total     int        `json:"total"`
	Values    []Schedule `json:"values,omitempty"`
	Schedules []Schedule `json:"schedules,omitempty"`
}

// ListSchedulesOptions controls schedule list request.
type ListSchedulesOptions struct {
	TeamID string
	Limit  int
	Offset int
}

// OnCallResponder is a minimal on-call responder DTO.
type OnCallResponder struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
	Type string `json:"type,omitempty"`
}

// OnCallRespondersResult wraps responders response.
type OnCallRespondersResult struct {
	Responders []OnCallResponder `json:"responders,omitempty"`
	Values     []OnCallResponder `json:"values,omitempty"`
}

// ListOnCallRespondersOptions controls responder listing.
type ListOnCallRespondersOptions struct {
	From  string
	To    string
	Limit int
}

func (r AlertsListResult) items() []Alert {
	if len(r.Values) > 0 {
		return r.Values
	}
	return r.Alerts
}

func (r TeamsListResult) items() []Team {
	if len(r.Values) > 0 {
		return r.Values
	}
	return r.Teams
}

func (r SchedulesListResult) items() []Schedule {
	if len(r.Values) > 0 {
		return r.Values
	}
	return r.Schedules
}

func (r OnCallRespondersResult) items() []OnCallResponder {
	if len(r.Values) > 0 {
		return r.Values
	}
	return r.Responders
}
