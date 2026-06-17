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
	audio                      [][]byte
	killSignalingAfterHandshake bool
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
		_ = c.WriteJSON(map[string]any{
			"msg_type":    msgSignalingHandshakeResp,
			"status_code": 0,
			"media_server": map[string]any{
				"server_urls": map[string]any{"all": "ws://" + r.Host + "/media"},
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
