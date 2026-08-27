package rtms

import (
	"bytes"
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// fakeServer stands in for the Zoom RTMS signaling+media servers over httptest.
// killSignalingAfterHandshake closes the signaling socket right after handing out
// the media URL, to exercise partial-failure teardown.
type fakeServerOpts struct {
	audio                       [][]byte
	killSignalingAfterHandshake bool
	splitURLs                   bool   // advertise audio+chat urls instead of `all`
	chatStatus                  int    // /chat data-handshake status_code
	chatText                    string // one chat message pushed after the /chat handshake
}

func fakeRTMSServer(t *testing.T, opts fakeServerOpts) string {
	t.Helper()
	up := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}

	mux := http.NewServeMux()
	mux.HandleFunc("/signaling", func(w http.ResponseWriter, r *http.Request) {
		c, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer c.Close()
		if _, _, err := c.ReadMessage(); err != nil { // msg 1 handshake req
			return
		}
		urls := map[string]any{"all": "ws://" + r.Host + "/media"}
		if opts.splitURLs {
			urls = map[string]any{"audio": "ws://" + r.Host + "/media", "chat": "ws://" + r.Host + "/chat"}
		}
		_ = c.WriteJSON(map[string]any{
			"msg_type":    msgSignalingHandshakeResp,
			"status_code": 0,
			"media_server": map[string]any{
				"server_urls": urls,
			},
		})
		if opts.killSignalingAfterHandshake {
			return // drop signaling — session must tear down, not hang
		}
		for { // drain ready-ACK / keep-alive until the client goes away
			if _, _, err := c.ReadMessage(); err != nil {
				return
			}
		}
	})
	mux.HandleFunc("/media", func(w http.ResponseWriter, r *http.Request) {
		c, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer c.Close()
		if _, _, err := c.ReadMessage(); err != nil { // msg 3 data handshake req
			return
		}
		_ = c.WriteJSON(map[string]any{"msg_type": msgDataHandshakeResp, "status_code": 0})
		for _, pcm := range opts.audio {
			_ = c.WriteJSON(map[string]any{
				"msg_type": msgMediaAudio,
				"content": map[string]any{
					"data":      base64.StdEncoding.EncodeToString(pcm),
					"user_id":   42,
					"user_name": "Alice",
				},
			})
		}
		for {
			if _, _, err := c.ReadMessage(); err != nil {
				return
			}
		}
	})

	mux.HandleFunc("/chat", func(w http.ResponseWriter, r *http.Request) {
		c, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer c.Close()
		if _, _, err := c.ReadMessage(); err != nil { // msg 3 data handshake req
			return
		}
		_ = c.WriteJSON(map[string]any{"msg_type": msgDataHandshakeResp, "status_code": opts.chatStatus})
		if opts.chatStatus == 0 && opts.chatText != "" {
			_ = c.WriteJSON(map[string]any{
				"msg_type": msgMediaChat,
				"content":  map[string]any{"user_id": 7, "user_name": "Bob", "data": opts.chatText, "timestamp": 1700000000000},
			})
		}
		for {
			if _, _, err := c.ReadMessage(); err != nil {
				return
			}
		}
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return "ws://" + strings.TrimPrefix(srv.URL, "http://") + "/signaling"
}

// Happy path: full handshake, one audio frame reaches OnAudio, and a caller cancel
// makes Run return nil promptly.
func TestSession_LifecycleAndCleanTeardown(t *testing.T) {
	t.Parallel()
	pcm := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	signalingURL := fakeRTMSServer(t, fakeServerOpts{audio: [][]byte{pcm}})

	got := make(chan AudioFrame, 4)
	sess, err := NewSession(Config{
		ClientID: "c", ClientSecret: "s",
		MeetingUUID: "m", StreamID: "stream-123",
		SignalingURL: signalingURL,
		Handlers:     Handlers{OnAudio: func(f AudioFrame) { got <- f }},
	})
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	runErr := make(chan error, 1)
	go func() { runErr <- sess.Run(ctx) }()

	select {
	case f := <-got:
		if !bytes.Equal(f.PCM, pcm) || f.UserID != "42" || f.UserName != "Alice" {
			t.Fatalf("bad frame: %+v", f)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("OnAudio never fired")
	}

	cancel()
	select {
	case err := <-runErr:
		if err != nil {
			t.Fatalf("Run after clean cancel = %v, want nil", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Run did not return after cancel (hang)")
	}
}

// Partial failure: signaling dies after the handshake. Run must tear the session
// down and return promptly (a non-nil error), not block forever on the media loop.
func TestSession_SignalingDeathEndsSession(t *testing.T) {
	t.Parallel()
	signalingURL := fakeRTMSServer(t, fakeServerOpts{killSignalingAfterHandshake: true})

	sess, err := NewSession(Config{
		ClientID: "c", ClientSecret: "s",
		MeetingUUID: "m", StreamID: "stream-xyz",
		SignalingURL: signalingURL,
		Handlers:     Handlers{OnAudio: func(AudioFrame) {}},
	})
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}

	runErr := make(chan error, 1)
	go func() { runErr <- sess.Run(context.Background()) }()

	select {
	case err := <-runErr:
		if err == nil {
			t.Fatal("Run = nil, want non-nil error after signaling death")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Run did not return after signaling death (hang — partial-failure teardown broken)")
	}
}

// Split urls: audio and chat flow over their own sockets.
func TestSession_SplitSockets_ChatDelivered(t *testing.T) {
	t.Parallel()
	pcm := []byte{1, 2, 3, 4}
	signalingURL := fakeRTMSServer(t, fakeServerOpts{audio: [][]byte{pcm}, splitURLs: true, chatText: "see https://a.b/c"})

	gotAudio := make(chan AudioFrame, 2)
	gotChat := make(chan ChatMessage, 2)
	sess, err := NewSession(Config{
		ClientID: "c", ClientSecret: "s", MeetingUUID: "m", StreamID: "st",
		SignalingURL: signalingURL,
		Handlers: Handlers{
			OnAudio: func(f AudioFrame) { gotAudio <- f },
			OnChat:  func(m ChatMessage) { gotChat <- m },
		},
	})
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- sess.Run(ctx) }()

	for range 2 {
		select {
		case f := <-gotAudio:
			if !bytes.Equal(f.PCM, pcm) {
				t.Fatalf("audio frame: %v", f)
			}
		case m := <-gotChat:
			if m.Text != "see https://a.b/c" || m.UserName != "Bob" {
				t.Fatalf("chat message: %+v", m)
			}
		case <-time.After(3 * time.Second):
			t.Fatal("timed out waiting for audio+chat")
		}
	}
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Run did not return after cancel")
	}
}

// Chat handshake rejection must not take the recording down.
func TestSession_ChatRejected_AudioSurvives(t *testing.T) {
	t.Parallel()
	pcm := []byte{9, 9, 9, 9}
	signalingURL := fakeRTMSServer(t, fakeServerOpts{audio: [][]byte{pcm}, splitURLs: true, chatStatus: 14})

	gotAudio := make(chan AudioFrame, 2)
	sess, err := NewSession(Config{
		ClientID: "c", ClientSecret: "s", MeetingUUID: "m", StreamID: "st",
		SignalingURL: signalingURL,
		Handlers: Handlers{
			OnAudio: func(f AudioFrame) { gotAudio <- f },
			OnChat:  func(ChatMessage) {},
		},
	})
	if err != nil {
		t.Fatalf("NewSession: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- sess.Run(ctx) }()

	select {
	case f := <-gotAudio:
		if !bytes.Equal(f.PCM, pcm) {
			t.Fatalf("audio frame: %v", f)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("audio did not flow after chat rejection")
	}
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run must stay clean when only chat is rejected, got: %v", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Run did not return after cancel")
	}
}
