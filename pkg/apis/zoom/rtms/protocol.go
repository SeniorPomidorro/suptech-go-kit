package rtms

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"strings"
)

// RTMS message types (the msg_type field).
const (
	msgSignalingHandshakeReq  = 1
	msgSignalingHandshakeResp = 2
	msgDataHandshakeReq       = 3
	msgDataHandshakeResp      = 4
	msgClientReadyAck         = 7
	msgStreamStateUpdate      = 8
	msgSessionStateUpdate     = 9
	msgKeepAliveReq           = 12
	msgKeepAliveResp          = 13
	msgMediaAudio             = 14
	msgMediaTranscript        = 17
)

// media_type subscription bitmask.
const (
	mediaTypeAudio      = 1
	mediaTypeTranscript = 8
)

// handshakeSignature signs the RTMS handshake the way the media server expects:
//
//	hex(HMAC_SHA256(clientSecret, "clientID,meetingUUID,streamID"))
func handshakeSignature(clientID, clientSecret, meetingUUID, streamID string) string {
	mac := hmac.New(sha256.New, []byte(clientSecret))
	mac.Write([]byte(clientID + "," + meetingUUID + "," + streamID))
	return hex.EncodeToString(mac.Sum(nil))
}

// peekMsgType reads just the msg_type discriminator from a frame.
func peekMsgType(b []byte) int {
	var p struct {
		MsgType int `json:"msg_type"`
	}
	_ = json.Unmarshal(b, &p)
	return p.MsgType
}

// decodeMediaContent extracts the binary payload from the content field, which is
// either a base64 string or an object {data, user_id, user_name, ...}. user_id may
// arrive as a number or a string; it is returned as a string.
func decodeMediaContent(data []byte) (pcm []byte, userID, userName string) {
	var msg struct {
		Content json.RawMessage `json:"content"`
	}
	if err := json.Unmarshal(data, &msg); err != nil || len(msg.Content) == 0 {
		return nil, "", ""
	}
	// content as an object
	var obj struct {
		Data     string          `json:"data"`
		UserID   json.RawMessage `json:"user_id"`
		UserName string          `json:"user_name"`
	}
	if err := json.Unmarshal(msg.Content, &obj); err == nil && obj.Data != "" {
		b, _ := base64.StdEncoding.DecodeString(obj.Data)
		return b, jsonScalarToString(obj.UserID), obj.UserName
	}
	// content as a plain base64 string (mixed-stream mode)
	var str string
	if err := json.Unmarshal(msg.Content, &str); err == nil && str != "" {
		b, _ := base64.StdEncoding.DecodeString(str)
		return b, "", ""
	}
	return nil, "", ""
}

// jsonScalarToString coerces a raw JSON scalar (string or number) into a string.
func jsonScalarToString(r json.RawMessage) string {
	s := strings.TrimSpace(string(r))
	if s == "" || s == "null" {
		return ""
	}
	return strings.Trim(s, `"`)
}

// swap16 swaps the byte order of every 16-bit sample (big-endian <-> little-endian).
func swap16(b []byte) {
	for i := 0; i+1 < len(b); i += 2 {
		b[i], b[i+1] = b[i+1], b[i]
	}
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
