package ws

import (
	"time"

	"github.com/gorilla/websocket"
)

// Client represents a WS connection subscribed to an org.
type Client struct {
	hub   *Hub
	orgID string
	conn  *websocket.Conn
	send  chan []byte
}

const (
	writeWait  = 10 * time.Second
	pongWait   = 60 * time.Second
	pingPeriod = (pongWait * 9) / 10
)

// NewClient creates a client wrapper.
func NewClient(hub *Hub, orgID string, conn *websocket.Conn) *Client {
	return &Client{
		hub:   hub,
		orgID: orgID,
		conn:  conn,
		send:  make(chan []byte, 64),
	}
}

// ReadPump pumps messages from WS (just for keepalive, we ignore client msgs).
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
			break
		}
	}
}

// WritePump pumps messages to WS.
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
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
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
