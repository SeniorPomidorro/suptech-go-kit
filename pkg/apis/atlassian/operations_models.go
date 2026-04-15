package atlassian

// CreateAlertResponse is the response from creating an alert.
type CreateAlertResponse struct {
	Result    string  `json:"result,omitempty"`
	RequestID string  `json:"requestId,omitempty"`
	Took      float64 `json:"took,omitempty"`
}

// Alert is a Jira Operations alert DTO.
type Alert struct {
	ID              string         `json:"id,omitempty"`
	TinyID          string         `json:"tinyId,omitempty"`
	Message         string         `json:"message,omitempty"`
	Description     string         `json:"description,omitempty"`
	Entity          string         `json:"entity,omitempty"`
	Source          string         `json:"source,omitempty"`
	Status          string         `json:"status,omitempty"`
	Alias           string         `json:"alias,omitempty"`
	Priority        string         `json:"priority,omitempty"`
	Owner           string         `json:"owner,omitempty"`
	Tags            []string       `json:"tags,omitempty"`
	Actions         []string       `json:"actions,omitempty"`
	Responders      []Responder    `json:"responders,omitempty"`
	ExtraProperties map[string]any `json:"extraProperties,omitempty"`
	Acknowledged    bool           `json:"acknowledged,omitempty"`
	Seen            bool           `json:"seen,omitempty"`
	Snoozed         bool           `json:"snoozed,omitempty"`
	Count           int            `json:"count,omitempty"`
	CreatedAt       string         `json:"createdAt,omitempty"`
	UpdatedAt       string         `json:"updatedAt,omitempty"`
	LastOccurredAt  string         `json:"lastOccurredAt,omitempty"`
	SnoozedUntil    string         `json:"snoozedUntil,omitempty"`
	IntegrationType string         `json:"integrationType,omitempty"`
	IntegrationName string         `json:"integrationName,omitempty"`
}

// Responder identifies a responder for an alert.
type Responder struct {
	ID   string `json:"id,omitempty"`
	Type string `json:"type,omitempty"`
}

// AlertsListResult represents a paginated alerts response.
type AlertsListResult struct {
	Values []Alert `json:"values,omitempty"`
	Count  int64   `json:"count,omitempty"`
}

// ListAlertsOptions controls alert listing.
type ListAlertsOptions struct {
	Query  string
	Size   int
	Offset int
	Order  string
	Sort   string
}

// Team is a Jira Operations team DTO.
type Team struct {
	TeamID   string `json:"teamId,omitempty"`
	TeamName string `json:"teamName,omitempty"`
}

// TeamsListResult represents a team list response.
type TeamsListResult struct {
	PlatformTeams []Team `json:"platformTeams,omitempty"`
}

// Schedule is an operations schedule DTO.
type Schedule struct {
	ID          string     `json:"id,omitempty"`
	Name        string     `json:"name,omitempty"`
	Description string     `json:"description,omitempty"`
	Timezone    string     `json:"timezone,omitempty"`
	Enabled     bool       `json:"enabled,omitempty"`
	TeamID      string     `json:"teamId,omitempty"`
	Rotations   []Rotation `json:"rotations,omitempty"`
}

// Rotation represents a schedule rotation.
type Rotation struct {
	ID           string          `json:"id,omitempty"`
	Name         string          `json:"name,omitempty"`
	StartDate    string          `json:"startDate,omitempty"`
	EndDate      string          `json:"endDate,omitempty"`
	Type         string          `json:"type,omitempty"`
	Length       int             `json:"length,omitempty"`
	Participants []ResponderInfo `json:"participants,omitempty"`
}

// ResponderInfo identifies a participant in a rotation.
type ResponderInfo struct {
	ID   string `json:"id,omitempty"`
	Type string `json:"type,omitempty"`
}

// SchedulesListResult represents paginated schedules.
type SchedulesListResult struct {
	Values []Schedule `json:"values,omitempty"`
}

// ListSchedulesOptions controls schedule list request.
type ListSchedulesOptions struct {
	Query  string
	Size   int
	Offset int
}

// OnCallParticipant represents a participant in the on-call tree.
type OnCallParticipant struct {
	ID                 string              `json:"id,omitempty"`
	Type               string              `json:"type,omitempty"`
	OnCallParticipants []OnCallParticipant `json:"onCallParticipants,omitempty"`
}

// OnCallResult wraps on-call response.
type OnCallResult struct {
	OnCallParticipants []OnCallParticipant `json:"onCallParticipants,omitempty"`
	OnCallUsers        []string            `json:"onCallUsers,omitempty"`
}

// ListOnCallOptions controls on-call listing.
type ListOnCallOptions struct {
	Flat bool
	Date string
}

// NotificationRule represents an operations notification rule.
type NotificationRule struct {
	ID         string `json:"id,omitempty"`
	Name       string `json:"name,omitempty"`
	ActionType string `json:"actionType,omitempty"`
	Enabled    bool   `json:"enabled,omitempty"`
	Order      int    `json:"order,omitempty"`
}

// NotificationRulesResult represents paginated notification rules.
type NotificationRulesResult struct {
	Values []NotificationRule `json:"values,omitempty"`
}

// ListNotificationRulesOptions controls notification rules listing.
type ListNotificationRulesOptions struct {
	Size   int
	Offset int
}
