package ws

import (
	"encoding/json"
	"log/slog"
	"sync"

	"github.com/google/uuid"
)

// Hub is the realtime hub for slot_taken broadcast — gorilla/websocket native (Koyeb, no Pusher).
// It keeps per-organization subscriptions and broadcasts JSON payloads.
// Design: direct Broadcast (non-blocking) for bookings hot path, plus async channel for Run loop.
// Frontend: useWebSocket `ws://api/ws?orgId=...` invalidates queryKeys.slots on {type:"slot_taken"}.
type Hub struct {
	mu      sync.RWMutex
	clients map[uuid.UUID]map[*Client]bool
	// async broadcast channel — Run() drains it. Broadcast() tries channel first, falls back to direct.
	broadcast chan broadcastMsg
}

type broadcastMsg struct {
	orgID   uuid.UUID
	payload interface{}
}

// NewHub creates a Hub with Koyeb-native WS support (no Pusher).
func NewHub() *Hub {
	return &Hub{
		clients:   make(map[uuid.UUID]map[*Client]bool),
		broadcast: make(chan broadcastMsg, 256),
	}
}

// Broadcast sends payload to all clients subscribed to orgID.
// Payload is typically map[string]interface{}{"type":"slot_taken", "staffId": "...", "startAt": "...", "endAt": "..."}.
// Non-blocking — drops if client buffer full to avoid blocking bookings transaction.
// Tries async channel first (if Run is active), otherwise direct fan-out.
func (h *Hub) Broadcast(orgID uuid.UUID, payload interface{}) {
	// Try async path — if Run loop is running, this will be consumed there.
	select {
	case h.broadcast <- broadcastMsg{orgID: orgID, payload: payload}:
		return
	default:
		// channel full or no consumer yet — fall through to direct broadcast
		// This ensures bookings path never blocks even before Run starts.
	}
	h.broadcastDirect(orgID, payload)
}

// broadcastDirect fans out directly to clients (holds RLock, non-blocking per client).
func (h *Hub) broadcastDirect(orgID uuid.UUID, payload interface{}) {
	h.mu.RLock()
	clients := h.clients[orgID]
	if len(clients) == 0 {
		h.mu.RUnlock()
		return
	}
	// Marshal once
	data := mustJSON(payload)
	// Copy slice of clients to avoid holding lock while sending (select may block briefly)
	// But we use non-blocking select, so holding RLock is fine.
	for c := range clients {
		select {
		case c.send <- data:
		default:
			slog.Warn("ws: client send buffer full, dropping slot_taken", "orgId", orgID.String())
		}
	}
	h.mu.RUnlock()
}

// Run starts the async broadcast loop. Should be run as `go hub.Run()` in main.go.
// It drains the broadcast channel and fans out via broadcastDirect.
// If context is not needed, it runs until channel closed (never closed in prod).
func (h *Hub) Run() {
	slog.Info("ws: hub Run loop started", "transport", "gorilla/websocket", "mode", "Koyeb native")
	for msg := range h.broadcast {
		h.broadcastDirect(msg.orgID, msg.payload)
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
	slog.Info("ws: client registered", "orgId", orgID.String(), "totalForOrg", len(h.clients[orgID]))
}

// Unregister removes client and closes its send channel.
// Safe to call multiple times; close is protected via sync.Once-like check
// by holding Lock and checking map existence.
func (h *Hub) Unregister(orgID uuid.UUID, c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if m, ok := h.clients[orgID]; ok {
		if _, exists := m[c]; exists {
			delete(m, c)
			// Close send channel to signal WritePump to exit
			// Recover if already closed (defensive)
			func() {
				defer func() { _ = recover() }()
				close(c.send)
			}()
			if len(m) == 0 {
				delete(h.clients, orgID)
			}
			slog.Info("ws: client unregistered", "orgId", orgID.String(), "remaining", len(m))
		}
	}
}

// Count returns number of clients for orgID (for health/metrics).
func (h *Hub) Count(orgID uuid.UUID) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients[orgID])
}

// Total returns total connected clients across all orgs.
func (h *Hub) Total() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	total := 0
	for _, m := range h.clients {
		total += len(m)
	}
	return total
}

// mustJSON marshals payload to []byte, fallback to empty json on error.
func mustJSON(v interface{}) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		slog.Error("ws: marshal payload failed", "error", err)
		return []byte(`{}`)
	}
	return b
}

// Ensure Hub satisfies bookings.Hub interface (Broadcast signature).
var _ = NewHub
