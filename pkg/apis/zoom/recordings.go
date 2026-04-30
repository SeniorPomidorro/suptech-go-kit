package zoom

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// RecordingFile types per spec recording_files[].file_type.
const (
	RecordingFileTypeMP4        = "MP4"        // video
	RecordingFileTypeM4A        = "M4A"        // audio
	RecordingFileTypeChat       = "CHAT"       // chat log
	RecordingFileTypeTranscript = "TRANSCRIPT" // VTT transcript
	RecordingFileTypeCC         = "CC"         // closed captions
	RecordingFileTypeCSV        = "CSV"        // poll/Q&A export
)

// RecordingFile represents one file produced by Zoom Cloud Recording.
// Subset from ZoomMeetings.json — recording_files[] schema.
type RecordingFile struct {
	ID             string `json:"id"`
	MeetingID      string `json:"meeting_id"`
	RecordingStart string `json:"recording_start"`
	RecordingEnd   string `json:"recording_end"`
	FileType       string `json:"file_type"`
	FileExtension  string `json:"file_extension"`
	FileSize       int64  `json:"file_size"`
	PlayURL        string `json:"play_url"`
	DownloadURL    string `json:"download_url"`
	Status         string `json:"status"`
	RecordingType  string `json:"recording_type"`
}

// MeetingRecordings is the response of GET /meetings/{meetingId}/recordings.
type MeetingRecordings struct {
	AccountID      string          `json:"account_id"`
	UUID           string          `json:"uuid"`
	ID             int64           `json:"id"`
	HostID         string          `json:"host_id"`
	Topic          string          `json:"topic"`
	StartTime      string          `json:"start_time"`
	Duration       int             `json:"duration"`
	TotalSize      int64           `json:"total_size"`
	RecordingCount int             `json:"recording_count"`
	RecordingFiles []RecordingFile `json:"recording_files"`
}

// UserRecordingsList is the paginated response of GET /users/{userId}/recordings.
type UserRecordingsList struct {
	From          string              `json:"from"`
	To            string              `json:"to"`
	PageSize      int                 `json:"page_size"`
	NextPageToken string              `json:"next_page_token"`
	Meetings      []MeetingRecordings `json:"meetings"`
}

// ListUserRecordingsOptions controls pagination and filtering for ListUser.
type ListUserRecordingsOptions struct {
	PageSize      int    // default 30, max 300
	NextPageToken string // empty = first page
	From          string // YYYY-MM-DD
	To            string // YYYY-MM-DD
	MeetingID     int64  // 0 = no filter
}

// RecordingsService groups cloud recording endpoints.
type RecordingsService struct {
	client *Client
}

// GetMeeting returns all recording files (and transcript, if present) for a meeting.
//
// Endpoint: GET /meetings/{meetingId}/recordings.
// Required scope (granular): cloud_recording:read:list_recording_files:admin.
func (s *RecordingsService) GetMeeting(ctx context.Context, meetingID int64) (*MeetingRecordings, error) {
	path := fmt.Sprintf("/meetings/%d/recordings", meetingID)
	var out MeetingRecordings
	if err := s.client.doJSON(ctx, http.MethodGet, path, nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// ListUser returns a page of recordings owned by the user.
//
// Endpoint: GET /users/{userId}/recordings.
// Required scope (granular): cloud_recording:read:list_user_recordings:admin.
func (s *RecordingsService) ListUser(ctx context.Context, userIDOrEmail string, opts *ListUserRecordingsOptions) (*UserRecordingsList, error) {
	path := fmt.Sprintf("/users/%s/recordings", url.PathEscape(userIDOrEmail))
	q := url.Values{}
	if opts != nil {
		if opts.PageSize > 0 {
			q.Set("page_size", strconv.Itoa(opts.PageSize))
		}
		if opts.NextPageToken != "" {
			q.Set("next_page_token", opts.NextPageToken)
		}
		if opts.From != "" {
			q.Set("from", opts.From)
		}
		if opts.To != "" {
			q.Set("to", opts.To)
		}
		if opts.MeetingID > 0 {
			q.Set("meeting_id", strconv.FormatInt(opts.MeetingID, 10))
		}
	}
	var out UserRecordingsList
	if err := s.client.doJSON(ctx, http.MethodGet, path, q, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
