package transport

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	coreerrors "github.com/AbolfazlZarei-dev/ParsRubika-bot-go/v3/core/errors"
)

// WebSocket wraps a gorilla/websocket.Conn and adds:
//
// - a background keep-alive ping loop
// - a mutex-protected write path (so multiple goroutines can send)
// - automatic reconnection support (managed by the caller)
type WebSocket struct {
	url     string
	conn    *websocket.Conn
	connMu  sync.Mutex
	headers map[string]string
}

// NewWebSocket constructs a WebSocket handle. The connection is not
// dialled yet — call Dial to open it.
func NewWebSocket(url string, headers map[string]string) *WebSocket {
	return &WebSocket{url: url, headers: headers}
}

// Dial opens the connection and returns the initial handshake response
// (so the caller can send the handShake frame). The headers supplied
// to NewWebSocket (User-Agent, auth, proxy, ...) are forwarded to the
// server as part of the HTTP upgrade request.
func (w *WebSocket) Dial(ctx context.Context) error {
	dialer := websocket.Dialer{
		TLSClientConfig:  &tls.Config{InsecureSkipVerify: false},
		HandshakeTimeout: 10 * time.Second,
	}
	header := http.Header{}
	for k, v := range w.headers {
		header.Set(k, v)
	}
	conn, _, err := dialer.DialContext(ctx, w.url, header)
	if err != nil {
		return coreerrors.Wrap(coreerrors.KindNetwork, "ws dial", err)
	}
	w.connMu.Lock()
	w.conn = conn
	w.connMu.Unlock()
	return nil
}

// SendJSON sends a JSON message over the WebSocket.
func (w *WebSocket) SendJSON(v any) error {
	w.connMu.Lock()
	defer w.connMu.Unlock()
	if w.conn == nil {
		return coreerrors.New(coreerrors.KindNetwork, "ws not connected")
	}
	return w.conn.WriteJSON(v)
}

// Read blocks until the next message arrives. It returns the raw bytes
// (text frames are already decoded; binary frames are passed as-is).
func (w *WebSocket) Read() ([]byte, error) {
	w.connMu.Lock()
	conn := w.conn
	w.connMu.Unlock()
	if conn == nil {
		return nil, coreerrors.New(coreerrors.KindNetwork, "ws not connected")
	}
	_, msg, err := conn.ReadMessage()
	if err != nil {
		return nil, err
	}
	return msg, nil
}

// Close terminates the WebSocket connection.
func (w *WebSocket) Close() error {
	w.connMu.Lock()
	defer w.connMu.Unlock()
	if w.conn == nil {
		return nil
	}
	err := w.conn.Close()
	w.conn = nil
	return err
}

// KeepAlive sends an empty JSON object every `interval`. It blocks
// until `ctx` is cancelled.
func (w *WebSocket) KeepAlive(ctx context.Context, interval time.Duration, onTick func() error) error {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := w.SendJSON(map[string]any{}); err != nil {
				return err
			}
			if onTick != nil {
				if err := onTick(); err != nil {
					return err
				}
			}
		}
	}
}

// Handshake sends the initial handShake frame required by the server.
func (w *WebSocket) Handshake(auth string) error {
	return w.SendJSON(map[string]any{
		"method":      "handShake",
		"auth":        auth,
		"api_version": "6",
		"data":        "",
	})
}

// DecodeMessage parses a WebSocket text frame into a generic map.
func DecodeMessage(raw []byte) (map[string]any, error) {
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("transport: ws decode: %w", err)
	}
	return m, nil
}
