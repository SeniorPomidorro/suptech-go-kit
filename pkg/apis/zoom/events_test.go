package zoom

import (
	"encoding/json"
	"testing"
)

// Zoom delivers fields flat in payload (meeting_id is a number); parts of the docs
// nest them under object. Both must decode, and the meeting_id number must not break it.
func TestRTMSEventPayload_FlatAndNested(t *testing.T) {
	t.Parallel()

	flat := `{"account_id":"a","meeting_uuid":"SM3gEV6kSJGUNOTwy2czcQ==","rtms_stream_id":"sid","server_urls":"wss://x","meeting_id":81626007526,"is_original_host":true}`
	var p1 RTMSEventPayload
	if err := json.Unmarshal([]byte(flat), &p1); err != nil {
		t.Fatalf("flat unmarshal: %v", err)
	}
	if f := p1.Fields(); f.MeetingUUID != "SM3gEV6kSJGUNOTwy2czcQ==" || f.RTMSStreamID != "sid" || f.ServerURLs != "wss://x" || f.MeetingID != 81626007526 {
		t.Errorf("flat fields wrong: %+v", f)
	}

	nested := `{"account_id":"a","object":{"meeting_uuid":"u2","rtms_stream_id":"sid2","server_urls":"wss://y"}}`
	var p2 RTMSEventPayload
	if err := json.Unmarshal([]byte(nested), &p2); err != nil {
		t.Fatalf("nested unmarshal: %v", err)
	}
	if f := p2.Fields(); f.RTMSStreamID != "sid2" || f.ServerURLs != "wss://y" {
		t.Errorf("nested fields wrong: %+v", f)
	}
}

// A meeting_id that arrives quoted (or as garbage) must not nuke the rest of the
// payload — the connection fields are what actually matter for starting a stream.
func TestRTMSEventPayload_MeetingIDTolerant(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		json string
		want MeetingID
	}{
		{"number", `{"meeting_id":81626007526,"rtms_stream_id":"sid","server_urls":"wss://x"}`, 81626007526},
		{"quoted string", `{"meeting_id":"81626007526","rtms_stream_id":"sid","server_urls":"wss://x"}`, 81626007526},
		{"garbage", `{"meeting_id":"not-a-number","rtms_stream_id":"sid","server_urls":"wss://x"}`, 0},
		{"null", `{"meeting_id":null,"rtms_stream_id":"sid","server_urls":"wss://x"}`, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var p RTMSEventPayload
			if err := json.Unmarshal([]byte(tc.json), &p); err != nil {
				t.Fatalf("unmarshal must not fail on meeting_id=%s: %v", tc.name, err)
			}
			f := p.Fields()
			if f.RTMSStreamID != "sid" || f.ServerURLs != "wss://x" {
				t.Errorf("connection fields lost: %+v", f)
			}
			if f.MeetingID != tc.want {
				t.Errorf("meeting_id = %d, want %d", f.MeetingID, tc.want)
			}
		})
	}
}

func TestURLValidationPlainToken(t *testing.T) {
	t.Parallel()
	if got := URLValidationPlainToken(json.RawMessage(`{"plainToken":"abc123"}`)); got != "abc123" {
		t.Fatalf("want abc123, got %q", got)
	}
	if got := URLValidationPlainToken(json.RawMessage(`{}`)); got != "" {
		t.Fatalf("want empty, got %q", got)
	}
}
