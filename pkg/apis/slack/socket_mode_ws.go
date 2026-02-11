package slack

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha1"
	"crypto/tls"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	webSocketHandshakeTimeout = 10 * time.Second
	maxWebSocketFrameSize     = 32 << 20 // 32 MB

	wsOpcodeContinuation = 0x0
	wsOpcodeText         = 0x1
	wsOpcodeBinary       = 0x2
	wsOpcodeClose        = 0x8
	wsOpcodePing         = 0x9
	wsOpcodePong         = 0xA
)

type rfc6455Dialer struct{}

func (d *rfc6455Dialer) Dial(ctx context.Context, wsURL string) (SocketModeConn, error) {
	endpoint, err := url.Parse(wsURL)
	if err != nil {
		return nil, fmt.Errorf("slack: parse websocket URL: %w", err)
	}
	if endpoint.Scheme != "ws" && endpoint.Scheme != "wss" {
		return nil, fmt.Errorf("slack: unsupported websocket scheme %q", endpoint.Scheme)
	}

	hostPort := endpoint.Host
	if !strings.Contains(hostPort, ":") {
		if endpoint.Scheme == "wss" {
			hostPort += ":443"
		} else {
			hostPort += ":80"
		}
	}

	dialer := &net.Dialer{}
	rawConn, err := dialer.DialContext(ctx, "tcp", hostPort)
	if err != nil {
		return nil, fmt.Errorf("slack: dial websocket host: %w", err)
	}

	conn := rawConn
	if endpoint.Scheme == "wss" {
		tlsConn := tls.Client(rawConn, &tls.Config{
			ServerName: endpoint.Hostname(),
			MinVersion: tls.VersionTLS12,
		})
		if err := tlsConn.HandshakeContext(ctx); err != nil {
			_ = rawConn.Close()
			return nil, fmt.Errorf("slack: tls handshake failed: %w", err)
		}
		conn = tlsConn
	}

	socketConn, err := websocketClientHandshake(ctx, conn, endpoint)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	return socketConn, nil
}

func websocketClientHandshake(ctx context.Context, conn net.Conn, endpoint *url.URL) (*websocketConn, error) {
	deadline := time.Now().Add(webSocketHandshakeTimeout)
	if d, ok := ctx.Deadline(); ok {
		deadline = d
	}
	if err := conn.SetDeadline(deadline); err != nil {
		return nil, fmt.Errorf("slack: set handshake deadline: %w", err)
	}
	defer func() { _ = conn.SetDeadline(time.Time{}) }()

	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("slack: generate websocket nonce: %w", err)
	}
	secWebSocketKey := base64.StdEncoding.EncodeToString(nonce)

	requestURI := endpoint.RequestURI()
	if requestURI == "" {
		requestURI = "/"
	}
	request := fmt.Sprintf(
		"GET %s HTTP/1.1\r\nHost: %s\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Key: %s\r\nSec-WebSocket-Version: 13\r\nUser-Agent: suptech-go-kit/socket-mode\r\n\r\n",
		requestURI,
		endpoint.Host,
		secWebSocketKey,
	)

	if _, err := io.WriteString(conn, request); err != nil {
		return nil, fmt.Errorf("slack: send websocket handshake: %w", err)
	}

	reader := bufio.NewReader(conn)
	resp, err := http.ReadResponse(reader, &http.Request{Method: http.MethodGet, URL: endpoint})
	if err != nil {
		return nil, fmt.Errorf("slack: read websocket handshake response: %w", err)
	}
	if resp.StatusCode != http.StatusSwitchingProtocols {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("slack: websocket handshake failed status=%d body=%q", resp.StatusCode, string(body))
	}
	expectedAccept := wsAcceptKey(secWebSocketKey)
	if resp.Header.Get("Sec-WebSocket-Accept") != expectedAccept {
		return nil, errors.New("slack: websocket handshake failed: invalid Sec-WebSocket-Accept")
	}

	return &websocketConn{
		conn:   conn,
		reader: reader,
	}, nil
}

func wsAcceptKey(secWebSocketKey string) string {
	hash := sha1.Sum([]byte(secWebSocketKey + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"))
	return base64.StdEncoding.EncodeToString(hash[:])
}

type websocketConn struct {
	conn   net.Conn
	reader *bufio.Reader

	writeMu sync.Mutex
}

func (c *websocketConn) ReadJSON(v any) error {
	message, err := c.readMessage()
	if err != nil {
		return err
	}
	if err := json.Unmarshal(message, v); err != nil {
		return fmt.Errorf("slack: decode websocket message: %w", err)
	}
	return nil
}

func (c *websocketConn) WriteJSON(v any) error {
	payload, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("slack: encode websocket message: %w", err)
	}
	return c.writeFrame(wsOpcodeText, payload)
}

func (c *websocketConn) Close() error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	if c.conn == nil {
		return nil
	}
	conn := c.conn
	c.conn = nil
	return conn.Close()
}

func (c *websocketConn) readMessage() ([]byte, error) {
	var (
		buffer             bytes.Buffer
		expectContinuation bool
	)

	for {
		opcode, fin, payload, err := c.readFrame()
		if err != nil {
			return nil, err
		}

		switch opcode {
		case wsOpcodeText, wsOpcodeBinary:
			if expectContinuation {
				return nil, errors.New("slack: unexpected websocket data frame")
			}
			if fin {
				return payload, nil
			}
			buffer.Write(payload)
			expectContinuation = true
		case wsOpcodeContinuation:
			if !expectContinuation {
				return nil, errors.New("slack: unexpected websocket continuation frame")
			}
			buffer.Write(payload)
			if fin {
				return buffer.Bytes(), nil
			}
		case wsOpcodePing:
			if err := c.writeFrame(wsOpcodePong, payload); err != nil {
				return nil, err
			}
		case wsOpcodePong:
			continue
		case wsOpcodeClose:
			return nil, io.EOF
		default:
			return nil, fmt.Errorf("slack: unsupported websocket opcode=%d", opcode)
		}
	}
}

func (c *websocketConn) readFrame() (byte, bool, []byte, error) {
	var header [2]byte
	if _, err := io.ReadFull(c.reader, header[:]); err != nil {
		return 0, false, nil, err
	}

	fin := header[0]&0x80 != 0
	opcode := header[0] & 0x0F

	maskBit := header[1]&0x80 != 0
	payloadLen, err := c.readPayloadLength(header[1] & 0x7F)
	if err != nil {
		return 0, false, nil, err
	}
	if payloadLen > maxWebSocketFrameSize {
		return 0, false, nil, fmt.Errorf("slack: websocket frame too large: %d", payloadLen)
	}

	var mask [4]byte
	if maskBit {
		if _, err := io.ReadFull(c.reader, mask[:]); err != nil {
			return 0, false, nil, err
		}
	}

	payload := make([]byte, payloadLen)
	if payloadLen > 0 {
		if _, err := io.ReadFull(c.reader, payload); err != nil {
			return 0, false, nil, err
		}
	}
	if maskBit {
		for i := range payload {
			payload[i] ^= mask[i%4]
		}
	}

	return opcode, fin, payload, nil
}

func (c *websocketConn) readPayloadLength(base byte) (int, error) {
	switch base {
	case 126:
		var extended [2]byte
		if _, err := io.ReadFull(c.reader, extended[:]); err != nil {
			return 0, err
		}
		return int(binary.BigEndian.Uint16(extended[:])), nil
	case 127:
		var extended [8]byte
		if _, err := io.ReadFull(c.reader, extended[:]); err != nil {
			return 0, err
		}
		length := binary.BigEndian.Uint64(extended[:])
		if length > uint64(^uint(0)>>1) {
			return 0, errors.New("slack: websocket payload length overflow")
		}
		return int(length), nil
	default:
		return int(base), nil
	}
}

func (c *websocketConn) writeFrame(opcode byte, payload []byte) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	if c.conn == nil {
		return io.ErrClosedPipe
	}

	frame, err := buildClientFrame(opcode, payload)
	if err != nil {
		return err
	}
	if _, err := c.conn.Write(frame); err != nil {
		return fmt.Errorf("slack: write websocket frame: %w", err)
	}
	return nil
}

func buildClientFrame(opcode byte, payload []byte) ([]byte, error) {
	var frame bytes.Buffer

	frame.WriteByte(0x80 | opcode)

	payloadLen := len(payload)
	switch {
	case payloadLen < 126:
		frame.WriteByte(byte(payloadLen) | 0x80)
	case payloadLen <= 0xFFFF:
		frame.WriteByte(126 | 0x80)
		var size [2]byte
		binary.BigEndian.PutUint16(size[:], uint16(payloadLen))
		frame.Write(size[:])
	default:
		frame.WriteByte(127 | 0x80)
		var size [8]byte
		binary.BigEndian.PutUint64(size[:], uint64(payloadLen))
		frame.Write(size[:])
	}

	mask := make([]byte, 4)
	if _, err := rand.Read(mask); err != nil {
		return nil, fmt.Errorf("slack: generate websocket mask: %w", err)
	}
	frame.Write(mask)

	maskedPayload := make([]byte, payloadLen)
	for i := range payload {
		maskedPayload[i] = payload[i] ^ mask[i%4]
	}
	frame.Write(maskedPayload)

	return frame.Bytes(), nil
}
