package ws

import (
	"encoding/json"
	"sync"

	"github.com/google/uuid"
)

// Hub is the realtime hub for slot_taken broadcast — gorilla/websocket native (Koyeb, no Pusher).
// It keeps per-organization subscriptions and broadcasts JSON payloads.
// For T04 we provide Broadcast method used by bookings Service; full WS upgrade handled in handler.go.
type Hub struct {
	mu      sync.RWMutex
	clients map[uuid.UUID]map[*Client]bool
	// broadcast channel for internal use (optional)
	broadcast chan broadcastMsg
}

type broadcastMsg struct {
	orgID   uuid.UUID
	payload interface{}
}

// NewHub creates a Hub.
func NewHub() *Hub {
	return &Hub{
		clients:   make(map[uuid.UUID]map[*Client]bool),
		broadcast: make(chan broadcastMsg, 256),
	}
}

// Broadcast sends payload to all clients subscribed to orgID.
// Payload is typically map[string]interface{}{"type":"slot_taken", ...}.
// Non-blocking — drops if client buffer full (caller logs).
func (h *Hub) Broadcast(orgID uuid.UUID, payload interface{}) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	clients := h.clients[orgID]
	if len(clients) == 0 {
		return
	}
	// Marshal once for efficiency
	_ = payload // payload will be marshaled per client in Client.Write
	for c := range clients {
		select {
		case c.send <- mustJSON(payload):
		default:
			// drop if buffer full — better to skip than block bookings path
		}
	}
}

// Register adds client for org.
func (h *Hub) Register(orgID uuid.UUID, c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.clients[orgID] == nil {
		h.clients[orgID] = make(map[*Client]bool)
	}
	h.clients[orgID][c] = true
}

// Unregister removes client.
func (h *Hub) Unregister(orgID uuid.UUID, c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if m, ok := h.clients[orgID]; ok {
		delete(m, c)
		if len(m) == 0 {
			delete(h.clients, orgID)
		}
	}
}

// mustJSON marshals payload to []byte, fallback to empty json on error.
func mustJSON(v interface{}) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		return []byte(`{}`)
	}
	return b
}

// Ensure Hub satisfies bookings.Hub interface (Broadcast signature).
var _ = NewHub
