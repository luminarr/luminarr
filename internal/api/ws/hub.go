// Package ws implements a WebSocket hub that bridges the internal event bus
// to connected browser clients.
package ws

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"nhooyr.io/websocket"

	"github.com/davidfic/luminarr/internal/events"
)

const (
	sendBufSize  = 32
	writeTimeout = 10 * time.Second
)

// Hub manages connected WebSocket clients and fans events out to them.
type Hub struct {
	mu      sync.Mutex
	clients map[chan []byte]struct{}
	apiKey  []byte
	logger  *slog.Logger
}

// NewHub creates a Hub that validates connections against apiKey.
func NewHub(apiKey string, logger *slog.Logger) *Hub {
	return &Hub{
		clients: make(map[chan []byte]struct{}),
		apiKey:  []byte(apiKey),
		logger:  logger,
	}
}

// HandleEvent implements events.Handler. It marshals the event to JSON and
// delivers it to every connected client. Slow clients are dropped (non-blocking
// send) to prevent one lagging browser from stalling all others.
func (h *Hub) HandleEvent(_ context.Context, e events.Event) {
	data, err := json.Marshal(e)
	if err != nil {
		h.logger.Error("ws: failed to marshal event", "error", err)
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.clients {
		select {
		case ch <- data:
		default:
			h.logger.Warn("ws: client send buffer full — dropping event")
		}
	}
}

// ServeHTTP upgrades the connection to WebSocket, validates the API key from
// the ?key= query parameter, and starts pumping events to the client.
func (h *Hub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Query().Get("key")
	if subtle.ConstantTimeCompare([]byte(key), h.apiKey) != 1 {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		// Skip the browser origin check. nhooyr.io/websocket rejects connections
		// where the Origin header doesn't match the Host header, which breaks the
		// Vite dev proxy (Origin: localhost:5173, Host: localhost:8282) and any
		// reverse-proxy setup. Auth is enforced by the API key check above.
		InsecureSkipVerify: true,
		CompressionMode:    websocket.CompressionContextTakeover,
	})
	if err != nil {
		h.logger.Error("ws: upgrade failed", "error", err)
		return
	}

	send := make(chan []byte, sendBufSize)

	h.mu.Lock()
	h.clients[send] = struct{}{}
	h.mu.Unlock()

	h.logger.Info("ws: client connected", "remote", r.RemoteAddr)

	defer func() {
		h.mu.Lock()
		delete(h.clients, send)
		h.mu.Unlock()
		close(send)
		h.logger.Info("ws: client disconnected", "remote", r.RemoteAddr)
	}()

	// readPump discards incoming frames and unblocks when the client closes.
	readDone := make(chan struct{})
	go func() {
		defer close(readDone)
		for {
			_, _, err := conn.Read(r.Context())
			if err != nil {
				return
			}
		}
	}()

	// writePump drains the send channel and forwards events to the client.
	for {
		select {
		case <-readDone:
			conn.Close(websocket.StatusNormalClosure, "")
			return
		case <-r.Context().Done():
			conn.Close(websocket.StatusGoingAway, "server shutting down")
			return
		case data, ok := <-send:
			if !ok {
				conn.Close(websocket.StatusNormalClosure, "")
				return
			}
			wCtx, cancel := context.WithTimeout(r.Context(), writeTimeout)
			err := conn.Write(wCtx, websocket.MessageText, data)
			cancel()
			if err != nil {
				h.logger.Debug("ws: write error", "error", err)
				return
			}
		}
	}
}
