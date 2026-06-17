// Package rtms is a client for Zoom Real-Time Media Streams (RTMS).
//
// Zoom ships no official Go SDK for RTMS, so this is a raw-WebSocket implementation
// of the documented protocol: a signaling socket (msg 1->2) negotiates a media
// server, a media socket (msg 3->4) subscribes to audio, and the client streams
// RAW L16 PCM (16 kHz mono) frames per participant (multi-stream, data_opt=2).
//
// The client is pure transport: it decodes audio frames and hands them to a
// Handlers.OnAudio callback. It knows nothing about transcription, storage, or
// what the caller does with the audio. Reference: developers.zoom.us/docs/rtms.
package rtms

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"

	"github.com/SeniorPomidorro/suptech-go-kit/pkg/transport"
)

// Audio format this client requests and therefore delivers: RAW L16 PCM, 16 kHz, mono.
const (
	SampleRateHz  = 16000
	Channels      = 1
	BitsPerSample = 16
)

// AudioFrame is one participant's audio chunk in multi-stream mode.
type AudioFrame struct {
	UserID   string
	UserName string
	PCM      []byte // RAW L16 PCM, little-endian (SampleRateHz / Channels / BitsPerSample)
}

// Transcript is Zoom's built-in transcript text, delivered only when the caller
// subscribes to the transcript media type (not enabled by this client by default).
type Transcript struct {
	UserName string
	Text     string
	IsFinal  bool
}

// Handlers receive session events. They are invoked from the media read goroutine
// and MUST NOT block — offload heavy work (transcription, disk, network) to your
// own goroutines, or you will stall audio reception and Zoom will drop the stream.
type Handlers struct {
	OnAudio      func(AudioFrame) // required
	OnTranscript func(Transcript) // optional
}

// Config describes one RTMS stream. SignalingURL, MeetingUUID and StreamID come
// from the meeting.rtms_started webhook; ClientID/ClientSecret are the Zoom app
// credentials used to sign the handshake.
type Config struct {
	ClientID     string
	ClientSecret string
	MeetingUUID  string
	StreamID     string
	SignalingURL string

	// ByteSwap swaps PCM byte order, for the rare case Zoom delivers big-endian L16.
	ByteSwap bool
	// Logger is optional; nil means silent.
	Logger transport.Logger

	Handlers Handlers
}

// Session is a single live RTMS connection (signaling + media sockets) for one stream.
type Session struct {
	cfg Config
	log transport.Logger

	sig   atomic.Pointer[safeConn]
	media atomic.Pointer[safeConn]
	wg    sync.WaitGroup

	cancel   context.CancelFunc
	failOnce sync.Once
	err      error
}

// NewSession validates cfg and returns a session ready to Run.
func NewSession(cfg Config) (*Session, error) {
	if cfg.ClientID == "" || cfg.ClientSecret == "" {
		return nil, errors.New("rtms: client id and client secret are required")
	}
	if cfg.MeetingUUID == "" || cfg.StreamID == "" || cfg.SignalingURL == "" {
		return nil, errors.New("rtms: meeting uuid, stream id and signaling url are required")
	}
	if cfg.Handlers.OnAudio == nil {
		return nil, errors.New("rtms: Handlers.OnAudio is required")
	}
	log := cfg.Logger
	if log == nil {
		log = nopLogger{}
	}
	return &Session{cfg: cfg, log: log}, nil
}

// Run connects and processes the stream until ctx is cancelled or the connection
// ends. It blocks. It returns the error that terminated the session, or nil when
// ctx was cancelled cleanly (the normal stop path).
func (s *Session) Run(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	s.cancel = cancel

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, s.cfg.SignalingURL, nil)
	if err != nil {
		return fmt.Errorf("rtms: signaling dial: %w", err)
	}
	s.sig.Store(newSafeConn(conn))

	// Close sockets on cancellation to unblock the read loops.
	go func() {
		<-ctx.Done()
		s.closeConns()
	}()

	handshake := map[string]any{
		"msg_type":         msgSignalingHandshakeReq,
		"protocol_version": 1,
		"meeting_uuid":     s.cfg.MeetingUUID,
		"rtms_stream_id":   s.cfg.StreamID,
		"sequence":         time.Now().UnixMilli(),
		"signature":        s.signature(),
		"media_type":       mediaTypeAudio,
	}
	if err := s.sig.Load().WriteJSON(handshake); err != nil {
		return fmt.Errorf("rtms: signaling handshake: %w", err)
	}
	s.log.Printf("rtms[%s]: signaling handshake sent", s.short())

	s.signalingLoop(ctx)
	s.wg.Wait()

	// fail() sets s.err and cancels; a caller-driven cancel leaves it nil — the
	// normal stop path, reported as success per the doc above.
	return s.err
}

func (s *Session) signalingLoop(ctx context.Context) {
	conn := s.sig.Load()
	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			// An unexpected signaling close is session-fatal: this socket carries
			// keep-alives and the ready-ACK, so tear the whole session down instead
			// of limping on with media only and blocking Run on wg.Wait.
			if ctx.Err() == nil {
				s.fail(fmt.Errorf("rtms: signaling closed: %w", err))
			}
			return
		}
		switch peekMsgType(data) {
		case msgSignalingHandshakeResp:
			var resp struct {
				StatusCode  int `json:"status_code"`
				MediaServer struct {
					ServerURLs struct {
						Audio string `json:"audio"`
						All   string `json:"all"`
					} `json:"server_urls"`
				} `json:"media_server"`
				MediaServerURL string `json:"media_server_url"` // flat fallback
			}
			_ = json.Unmarshal(data, &resp)
			if resp.StatusCode != 0 {
				s.fail(fmt.Errorf("rtms: signaling handshake rejected: status=%d", resp.StatusCode))
				return
			}
			mediaURL := firstNonEmpty(resp.MediaServer.ServerURLs.All, resp.MediaServer.ServerURLs.Audio, resp.MediaServerURL)
			if mediaURL == "" {
				s.fail(errors.New("rtms: no media server url in signaling response"))
				return
			}
			s.log.Printf("rtms[%s]: signaling OK, media server: %s", s.short(), mediaURL)
			s.wg.Add(1)
			go s.mediaLoop(ctx, mediaURL)

		case msgKeepAliveReq:
			s.replyKeepAlive(conn, data)

		case msgStreamStateUpdate, msgSessionStateUpdate:
			s.log.Printf("rtms[%s]: state update: %s", s.short(), string(data))

		default:
			s.log.Printf("rtms[%s]: signaling msg type=%d", s.short(), peekMsgType(data))
		}
	}
}

func (s *Session) mediaLoop(ctx context.Context, url string) {
	defer s.wg.Done()

	conn, _, err := websocket.DefaultDialer.DialContext(ctx, url, nil)
	if err != nil {
		s.fail(fmt.Errorf("rtms: media dial: %w", err))
		return
	}
	mc := newSafeConn(conn)
	s.media.Store(mc)

	// Data handshake: request RAW L16 PCM 16 kHz mono, multi-stream, 20 ms chunks.
	req := map[string]any{
		"msg_type":           msgDataHandshakeReq,
		"protocol_version":   1,
		"meeting_uuid":       s.cfg.MeetingUUID,
		"rtms_stream_id":     s.cfg.StreamID,
		"signature":          s.signature(),
		"media_type":         mediaTypeAudio,
		"payload_encryption": false,
		"media_params": map[string]any{
			"audio": map[string]any{
				"content_type": 2,  // RAW_AUDIO
				"sample_rate":  1,  // 16 kHz
				"channel":      1,  // mono
				"codec":        1,  // L16 (PCM)
				"data_opt":     2,  // AUDIO_MULTI_STREAMS (per-participant, with user_id)
				"send_rate":    20, // ms per chunk
			},
		},
	}
	if err := mc.WriteJSON(req); err != nil {
		s.fail(fmt.Errorf("rtms: media handshake: %w", err))
		return
	}
	s.log.Printf("rtms[%s]: media handshake sent", s.short())

	for {
		_, data, err := mc.ReadMessage()
		if err != nil {
			// Unexpected media close ends the session too — otherwise signaling
			// keeps Run alive with no audio flowing.
			if ctx.Err() == nil {
				s.fail(fmt.Errorf("rtms: media closed: %w", err))
			}
			return
		}
		switch peekMsgType(data) {
		case msgDataHandshakeResp:
			var resp struct {
				StatusCode int `json:"status_code"`
			}
			_ = json.Unmarshal(data, &resp)
			if resp.StatusCode != 0 {
				s.fail(fmt.Errorf("rtms: media handshake rejected: status=%d", resp.StatusCode))
				return
			}
			// Ready to receive — ACK over the signaling socket.
			if sc := s.sig.Load(); sc != nil {
				_ = sc.WriteJSON(map[string]any{
					"msg_type":       msgClientReadyAck,
					"meeting_uuid":   s.cfg.MeetingUUID,
					"rtms_stream_id": s.cfg.StreamID,
				})
			}
			s.log.Printf("rtms[%s]: media ready — multi-stream audio flowing", s.short())

		case msgMediaAudio:
			s.handleAudio(data)

		case msgMediaTranscript:
			s.handleTranscript(data)

		case msgKeepAliveReq:
			s.replyKeepAlive(mc, data)

		default:
			s.log.Printf("rtms[%s]: media msg type=%d", s.short(), peekMsgType(data))
		}
	}
}

func (s *Session) handleAudio(data []byte) {
	pcm, userID, userName := decodeMediaContent(data)
	if len(pcm) == 0 {
		return
	}
	if s.cfg.ByteSwap {
		swap16(pcm)
	}
	s.cfg.Handlers.OnAudio(AudioFrame{UserID: userID, UserName: userName, PCM: pcm})
}

func (s *Session) handleTranscript(data []byte) {
	if s.cfg.Handlers.OnTranscript == nil {
		return
	}
	var m struct {
		Content struct {
			Text     string `json:"text"`
			Data     string `json:"data"`
			UserName string `json:"user_name"`
			IsFinal  bool   `json:"is_final"`
		} `json:"content"`
	}
	if err := json.Unmarshal(data, &m); err != nil {
		return
	}
	text := firstNonEmpty(m.Content.Text, m.Content.Data)
	if text != "" {
		s.cfg.Handlers.OnTranscript(Transcript{UserName: m.Content.UserName, Text: text, IsFinal: m.Content.IsFinal})
	}
}

// replyKeepAlive echoes a keep-alive request. RTMS is server-initiated: the media
// server sends msg 12 and we reply msg 13 — the client does not originate keep-alives.
func (s *Session) replyKeepAlive(conn *safeConn, data []byte) {
	var ka struct {
		Timestamp int64 `json:"timestamp"`
	}
	_ = json.Unmarshal(data, &ka)
	_ = conn.WriteJSON(map[string]any{
		"msg_type":  msgKeepAliveResp,
		"timestamp": ka.Timestamp,
	})
}

func (s *Session) fail(err error) {
	s.failOnce.Do(func() { s.err = err })
	if s.cancel != nil {
		s.cancel()
	}
}

func (s *Session) closeConns() {
	if c := s.sig.Load(); c != nil {
		_ = c.Close()
	}
	if c := s.media.Load(); c != nil {
		_ = c.Close()
	}
}

func (s *Session) signature() string {
	return handshakeSignature(s.cfg.ClientID, s.cfg.ClientSecret, s.cfg.MeetingUUID, s.cfg.StreamID)
}

func (s *Session) short() string {
	id := s.cfg.StreamID
	if id == "" {
		return "????????"
	}
	if len(id) > 8 {
		return id[:8]
	}
	return id
}

// safeConn allows writes from multiple goroutines (keep-alive replies, ACKs).
type safeConn struct {
	*websocket.Conn
	wmu sync.Mutex
}

func newSafeConn(c *websocket.Conn) *safeConn { return &safeConn{Conn: c} }

func (c *safeConn) WriteJSON(v any) error {
	c.wmu.Lock()
	defer c.wmu.Unlock()
	return c.Conn.WriteJSON(v)
}

type nopLogger struct{}

func (nopLogger) Printf(string, ...any) {}
