package zoom

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// MeetingType values per ZoomMeetings.json schema for meeting "type":
// 1 = instant, 2 = scheduled, 3 = recurring no fixed time, 8 = recurring fixed time, 10 = screen share only.
type MeetingType int

const (
	MeetingTypeInstant   MeetingType = 1
	MeetingTypeScheduled MeetingType = 2
)

// AutoRecording values: "local", "cloud", "none". Reference: settings.auto_recording in spec.
const (
	AutoRecordingCloud = "cloud"
	AutoRecordingLocal = "local"
	AutoRecordingNone  = "none"
)

// CreateMeetingRequest is a minimal-but-typed body for POST /users/{userId}/meetings.
// Only fields we actually need are included. See docs/api-specs/ZoomMeetings.json
// (paths."/users/{userId}/meetings".post.requestBody) for the full schema.
type CreateMeetingRequest struct {
	Topic    string                `json:"topic,omitempty"`
	Type     MeetingType           `json:"type,omitempty"`
	Agenda   string                `json:"agenda,omitempty"`
	Password string                `json:"password,omitempty"`
	Duration int                   `json:"duration,omitempty"`
	Settings *CreateMeetingSetting `json:"settings,omitempty"`
}

// CreateMeetingSetting carries the subset of settings.* fields we use.
// Pointers for booleans so the omitempty distinction "not set" vs "explicitly false" is preserved.
type CreateMeetingSetting struct {
	JoinBeforeHost                  *bool  `json:"join_before_host,omitempty"`
	WaitingRoom                     *bool  `json:"waiting_room,omitempty"`
	UsePMI                          *bool  `json:"use_pmi,omitempty"`
	HostVideo                       *bool  `json:"host_video,omitempty"`
	ParticipantVideo                *bool  `json:"participant_video,omitempty"`
	MuteUponEntry                   *bool  `json:"mute_upon_entry,omitempty"`
	AutoRecording                   string `json:"auto_recording,omitempty"`
	ApprovalType                    *int   `json:"approval_type,omitempty"`
	AlternativeHosts                string `json:"alternative_hosts,omitempty"`
	AlternativeHostsEmailNotify     *bool  `json:"alternative_hosts_email_notification,omitempty"`
	MeetingAuthentication           *bool  `json:"meeting_authentication,omitempty"`
	AutoStartMeetingSummary         *bool  `json:"auto_start_meeting_summary,omitempty"`
	AutoStartAICompanionQuestions   *bool  `json:"auto_start_ai_companion_questions,omitempty"`
	AutoAddRecordingToVideoMgmt     *bool  `json:"auto_add_recording_to_video_management,omitempty"`
}

// Meeting is the response shape for POST/GET /meetings.
// Subset of fields from spec — extend as needed.
type Meeting struct {
	UUID         string `json:"uuid"`
	ID           int64  `json:"id"`
	HostID       string `json:"host_id"`
	HostEmail    string `json:"host_email"`
	Topic        string `json:"topic"`
	Type         int    `json:"type"`
	Status       string `json:"status"`
	StartTime    string `json:"start_time"`
	Duration     int    `json:"duration"`
	Timezone     string `json:"timezone"`
	Agenda       string `json:"agenda"`
	CreatedAt    string `json:"created_at"`
	StartURL     string `json:"start_url"`
	JoinURL      string `json:"join_url"`
	Password     string `json:"password"`
	PMI          int64  `json:"pmi"`
	Settings     map[string]any `json:"settings,omitempty"`
}

// PastMeeting is the response shape for GET /past_meetings/{meetingId}.
type PastMeeting struct {
	UUID              string `json:"uuid"`
	ID                int64  `json:"id"`
	HostID            string `json:"host_id"`
	UserName          string `json:"user_name"`
	UserEmail         string `json:"user_email"`
	Topic             string `json:"topic"`
	Type              int    `json:"type"`
	StartTime         string `json:"start_time"`
	EndTime           string `json:"end_time"`
	Duration          int    `json:"duration"`
	TotalMinutes      int    `json:"total_minutes"`
	ParticipantsCount int    `json:"participants_count"`
	Source            string `json:"source"`
	HasMeetingSummary bool   `json:"has_meeting_summary"`
}

// MeetingListType values per spec for the type query param of GET /users/{userId}/meetings.
const (
	MeetingListTypeScheduled        = "scheduled"
	MeetingListTypeLive             = "live"
	MeetingListTypeUpcoming         = "upcoming"
	MeetingListTypeUpcomingMeetings = "upcoming_meetings"
	MeetingListTypePreviousMeetings = "previous_meetings"
)

// ListMeetingsOptions controls filtering and pagination of GET /users/{userId}/meetings.
type ListMeetingsOptions struct {
	// Type filters by meeting state. Empty = scheduled (Zoom's default).
	// Use MeetingListTypeLive to discover whether a host has an active meeting.
	Type          string
	PageSize      int    // default 30, max 300
	NextPageToken string // empty = first page
	PageNumber    int    // alternative to NextPageToken
	From          string // YYYY-MM-DD, only for previous_meetings/upcoming_meetings
	To            string // YYYY-MM-DD
	Timezone      string
}

// ListMeetingsResponse is the paginated response of GET /users/{userId}/meetings.
type ListMeetingsResponse struct {
	NextPageToken string    `json:"next_page_token"`
	PageCount     int       `json:"page_count"`
	PageNumber    int       `json:"page_number"`
	PageSize      int       `json:"page_size"`
	TotalRecords  int       `json:"total_records"`
	Meetings      []Meeting `json:"meetings"`
}

// MeetingsService groups meeting-related endpoints.
type MeetingsService struct {
	client *Client
}

// Create schedules a meeting on behalf of the user identified by userIDOrEmail.
//
// Endpoint: POST /users/{userId}/meetings.
// Required scope (granular): meeting:write:meeting:admin.
func (s *MeetingsService) Create(ctx context.Context, userIDOrEmail string, req *CreateMeetingRequest) (*Meeting, error) {
	if strings.TrimSpace(userIDOrEmail) == "" {
		return nil, fmt.Errorf("zoom: userId is required")
	}
	path := fmt.Sprintf("/users/%s/meetings", url.PathEscape(userIDOrEmail))

	var out Meeting
	if err := s.client.doJSON(ctx, http.MethodPost, path, nil, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Get fetches meeting details by ID.
//
// Endpoint: GET /meetings/{meetingId}.
// Required scope (granular): meeting:read:meeting:admin.
func (s *MeetingsService) Get(ctx context.Context, meetingID int64) (*Meeting, error) {
	path := fmt.Sprintf("/meetings/%d", meetingID)
	var out Meeting
	if err := s.client.doJSON(ctx, http.MethodGet, path, nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetPast returns metadata of a finished meeting. Useful to detect that a
// meeting has ended (the endpoint only returns 200 once the meeting is over).
//
// Endpoint: GET /past_meetings/{meetingId}.
// Required scope (granular): meeting:read:past_meeting:admin.
func (s *MeetingsService) GetPast(ctx context.Context, meetingID int64) (*PastMeeting, error) {
	path := fmt.Sprintf("/past_meetings/%d", meetingID)
	var out PastMeeting
	if err := s.client.doJSON(ctx, http.MethodGet, path, nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Update patches an existing meeting. Pass only the fields to change.
//
// Endpoint: PATCH /meetings/{meetingId}.
// Required scope (granular): meeting:update:meeting:admin.
func (s *MeetingsService) Update(ctx context.Context, meetingID int64, req *CreateMeetingRequest) error {
	path := fmt.Sprintf("/meetings/%d", meetingID)
	return s.client.doJSON(ctx, http.MethodPatch, path, nil, req, nil)
}

// Delete removes a meeting.
//
// Endpoint: DELETE /meetings/{meetingId}.
// Required scope (granular): meeting:delete:meeting:admin.
func (s *MeetingsService) Delete(ctx context.Context, meetingID int64) error {
	path := fmt.Sprintf("/meetings/%d", meetingID)
	return s.client.doJSON(ctx, http.MethodDelete, path, nil, nil, nil)
}

// ListByUser returns meetings owned by a user, optionally filtered by state.
// Use opts.Type = MeetingListTypeLive to check whether a host currently has an
// active meeting — that's how we pick a free host from the pool without storing
// state ourselves.
//
// Endpoint: GET /users/{userId}/meetings.
// Required scope (granular): meeting:read:list_meetings:admin.
func (s *MeetingsService) ListByUser(ctx context.Context, userIDOrEmail string, opts *ListMeetingsOptions) (*ListMeetingsResponse, error) {
	if strings.TrimSpace(userIDOrEmail) == "" {
		return nil, fmt.Errorf("zoom: userId is required")
	}
	path := fmt.Sprintf("/users/%s/meetings", url.PathEscape(userIDOrEmail))

	q := url.Values{}
	if opts != nil {
		if opts.Type != "" {
			q.Set("type", opts.Type)
		}
		if opts.PageSize > 0 {
			q.Set("page_size", strconv.Itoa(opts.PageSize))
		}
		if opts.NextPageToken != "" {
			q.Set("next_page_token", opts.NextPageToken)
		}
		if opts.PageNumber > 0 {
			q.Set("page_number", strconv.Itoa(opts.PageNumber))
		}
		if opts.From != "" {
			q.Set("from", opts.From)
		}
		if opts.To != "" {
			q.Set("to", opts.To)
		}
		if opts.Timezone != "" {
			q.Set("timezone", opts.Timezone)
		}
	}

	var out ListMeetingsResponse
	if err := s.client.doJSON(ctx, http.MethodGet, path, q, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
