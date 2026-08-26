package rtms

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"testing"
)

// signature = hex(HMAC_SHA256(client_secret, "client_id,meeting_uuid,rtms_stream_id"))
func TestHandshakeSignature(t *testing.T) {
	t.Parallel()
	got := handshakeSignature("cid", "csecret", "uuid", "stream")
	mac := hmac.New(sha256.New, []byte("csecret"))
	mac.Write([]byte("cid,uuid,stream"))
	want := hex.EncodeToString(mac.Sum(nil))
	if got != want {
		t.Fatalf("signature = %q, want %q", got, want)
	}
}

// content arrives both as an object {data,...} and as a base64 string — support both.
func TestDecodeMediaContent(t *testing.T) {
	t.Parallel()
	raw := base64.StdEncoding.EncodeToString([]byte{1, 2, 3, 4})

	b, uid, who := decodeMediaContent([]byte(`{"msg_type":14,"content":{"data":"` + raw + `","user_id":123,"user_name":"Bob"}}`))
	if !bytes.Equal(b, []byte{1, 2, 3, 4}) || who != "Bob" || uid != "123" {
		t.Errorf("object form: got %v / uid=%q / who=%q", b, uid, who)
	}

	b2, _, _ := decodeMediaContent([]byte(`{"msg_type":14,"content":"` + raw + `"}`))
	if !bytes.Equal(b2, []byte{1, 2, 3, 4}) {
		t.Errorf("string form: got %v", b2)
	}

	if b3, _, _ := decodeMediaContent([]byte(`{"msg_type":12}`)); b3 != nil {
		t.Errorf("no content: got %v, want nil", b3)
	}
}

func TestSwap16(t *testing.T) {
	t.Parallel()
	b := []byte{0x01, 0x02, 0x03, 0x04}
	swap16(b)
	if !bytes.Equal(b, []byte{0x02, 0x01, 0x04, 0x03}) {
		t.Errorf("swap16 = %v", b)
	}
}

func TestNewSession_Validation(t *testing.T) {
	t.Parallel()
	good := Config{
		ClientID: "c", ClientSecret: "s", MeetingUUID: "m", StreamID: "id",
		SignalingURL: "wss://x", Handlers: Handlers{OnAudio: func(AudioFrame) {}},
	}
	if _, err := NewSession(good); err != nil {
		t.Fatalf("valid config rejected: %v", err)
	}

	missingAudio := good
	missingAudio.Handlers = Handlers{}
	if _, err := NewSession(missingAudio); err == nil {
		t.Error("expected error when OnAudio is nil")
	}

	missingCreds := good
	missingCreds.ClientSecret = ""
	if _, err := NewSession(missingCreds); err == nil {
		t.Error("expected error when client secret is empty")
	}
}

// chat frames carry plain UTF-8 text and scalar ids that may be numbers or strings.
func TestHandleChat(t *testing.T) {
	t.Parallel()
	var got []ChatMessage
	s := &Session{cfg: Config{Handlers: Handlers{OnChat: func(m ChatMessage) { got = append(got, m) }}}}

	s.handleChat([]byte(`{"msg_type":18,"content":{"user_id":123,"user_name":"Bob","data":"see https://a.b/c","timestamp":1700000000000,"message_id":"m1","operation_type":"send"}}`))
	s.handleChat([]byte(`{"msg_type":18,"content":{"user_name":"Eve","text":"hi"}}`))
	s.handleChat([]byte(`{"msg_type":18,"content":{"user_name":"Empty"}}`))
	s.handleChat([]byte(`not-json`))

	if len(got) != 2 {
		t.Fatalf("delivered %d messages, want 2", len(got))
	}
	m := got[0]
	if m.UserID != "123" || m.UserName != "Bob" || m.Text != "see https://a.b/c" || m.MessageID != "m1" || m.OperationType != "send" || m.Timestamp != 1700000000000 {
		t.Errorf("first message = %+v", m)
	}
	if got[1].Text != "hi" {
		t.Errorf("text fallback = %+v", got[1])
	}
}

func TestMediaTypesBitmask(t *testing.T) {
	t.Parallel()
	audioOnly := &Session{cfg: Config{Handlers: Handlers{OnAudio: func(AudioFrame) {}}}}
	if got := audioOnly.mediaTypes(); got != mediaTypeAudio {
		t.Fatalf("audio-only mask = %d, want %d", got, mediaTypeAudio)
	}
	withChat := &Session{cfg: Config{Handlers: Handlers{OnAudio: func(AudioFrame) {}, OnChat: func(ChatMessage) {}}}}
	if got := withChat.mediaTypes(); got != mediaTypeAudio|mediaTypeChat {
		t.Fatalf("chat mask = %d, want %d", got, mediaTypeAudio|mediaTypeChat)
	}
}
