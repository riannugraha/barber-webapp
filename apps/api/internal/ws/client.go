package ws

import (
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// Client represents a WS connection subscribed to an org.
// Each orgID has its own broadcast group — slot_taken is scoped per organization.
type Client struct {
	hub   *Hub
	orgID uuid.UUID
	conn  *websocket.Conn
	send  chan []byte
}

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10 // 54s
)

// NewClient creates a client wrapper with buffered send channel (256).
// Buffer 256 is enough for bursts of slot_taken broadcasts without blocking bookings path.
func NewClient(hub *Hub, orgID uuid.UUID, conn *websocket.Conn) *Client {
	return &Client{
		hub:   hub,
		orgID: orgID,
		conn:  conn,
		send:  make(chan []byte, 256),
	}
}

// ReadPump pumps messages from WS (keepalive, we ignore client msgs).
// It sets read deadline on pong and breaks on error. Deferred close handled by handler's Unregister.
func (c *Client) ReadPump() {
	defer c.conn.Close()
	c.conn.SetReadLimit(512)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})
	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			// Normal closure or timeout — log at debug
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				// Log is done via slog in hub if needed, but we keep silent to avoid spam
			}
			break
		}
		// We ignore client-sent messages — this is a broadcast-only channel (slot_taken).
		// If client sends ping-like JSON, we just keep reading to keep pong handler alive.
	}
}

// WritePump pumps messages to WS with periodic pings to keep Koyeb connection alive.
// Koyeb's proxy respects WS pings; we use gorilla's PingMessage every 54s.
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		_ = c.conn.Close()
	}()
	for {
		select {
		case msg, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Channel closed — hub Unregistered
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			// Use NextWriter for batching? Simple WriteMessage is fine for JSON slot_taken.
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
