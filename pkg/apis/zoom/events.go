package zoom

import (
	"bytes"
	"encoding/json"
	"strconv"
	"strings"
)

// Webhook event names this package helps handle.
const (
	EventURLValidation      = "endpoint.url_validation"
	EventMeetingRTMSStarted = "meeting.rtms_started"
	EventMeetingRTMSStopped = "meeting.rtms_stopped"
	EventWebinarRTMSStarted = "webinar.rtms_started"
	EventWebinarRTMSStopped = "webinar.rtms_stopped"
)

// WebhookEnvelope is the common wrapper around every Zoom webhook event.
type WebhookEnvelope struct {
	Event   string          `json:"event"`
	EventTS int64           `json:"event_ts"`
	Payload json.RawMessage `json:"payload"`
}

// MeetingID is Zoom's numeric meeting id. Most payloads send it as a JSON number,
// but some send it quoted; it decodes from either form and never fails the
// surrounding payload on a malformed value (it stays 0 instead) — losing the rest
// of an rtms_started event over a stray meeting_id would be far worse than a 0 id.
type MeetingID int64

func (m *MeetingID) UnmarshalJSON(b []byte) error {
	s := strings.TrimSpace(string(bytes.Trim(bytes.TrimSpace(b), `"`)))
	if s == "" || s == "null" {
		return nil
	}
	if n, err := strconv.ParseInt(s, 10, 64); err == nil {
		*m = MeetingID(n)
	}
	return nil
}

// RTMSFields are the connection details delivered by an rtms_started event.
type RTMSFields struct {
	MeetingUUID  string    `json:"meeting_uuid"`
	MeetingID    MeetingID `json:"meeting_id"` // numeric meeting id; matches the meeting to its incident
	RTMSStreamID string    `json:"rtms_stream_id"`
	ServerURLs   string    `json:"server_urls"` // signaling WS URL — a string, not an array
}

// RTMSEventPayload decodes an rtms_started/stopped payload. Real Zoom delivers the
// fields flat inside payload; parts of the docs and mock servers nest them under
// payload.object. Both shapes are supported via Fields.
type RTMSEventPayload struct {
	AccountID  string      `json:"account_id"`
	Object     *RTMSFields `json:"object"` // nested variant (when present)
	RTMSFields             // flat variant (real Zoom)
}

// Fields returns the RTMS connection details regardless of payload shape.
func (p *RTMSEventPayload) Fields() RTMSFields {
	if p.Object != nil && p.Object.RTMSStreamID != "" {
		return *p.Object
	}
	return p.RTMSFields
}

// URLValidationPlainToken extracts plainToken from an endpoint.url_validation payload.
// Returns "" if absent. Pair with AnswerURLValidation to build the challenge response.
func URLValidationPlainToken(payload json.RawMessage) string {
	var p struct {
		PlainToken string `json:"plainToken"`
	}
	_ = json.Unmarshal(payload, &p)
	return strings.TrimSpace(p.PlainToken)
}
