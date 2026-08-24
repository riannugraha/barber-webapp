package ws

import (
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// For production, CheckOrigin should validate AllowedOrigins.
	// MVP: allow all origins — Koyeb native WS needs permissive for Vercel preview domains.
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Handler handles GET /ws?orgId=...
type Handler struct {
	hub *Hub
}

// NewHandler creates WS handler with Koyeb-native Hub (no Pusher).
func NewHandler(hub *Hub) *Handler {
	return &Handler{hub: hub}
}

// RegisterRoutes mounts GET /ws under group (e.g., /api/v1).
// Frontend: useWebSocket(`ws://api/ws?orgId=...`) invalidates queryKeys.slots on slot_taken.
func (h *Handler) RegisterRoutes(g *echo.Group) {
	g.GET("/ws", h.Upgrade)
}

// Upgrade upgrades to WS and subscribes to orgId broadcasts.
// Query: orgId (UUID, required). Also accepts orgID/organizationId for flexibility.
// Returns 422 if missing/invalid, 101 on success.
func (h *Handler) Upgrade(c echo.Context) error {
	orgIDStr := c.QueryParam("orgId")
	if orgIDStr == "" {
		orgIDStr = c.QueryParam("orgID")
	}
	if orgIDStr == "" {
		orgIDStr = c.QueryParam("organizationId")
	}
	if orgIDStr == "" {
		orgIDStr = c.QueryParam("organization_id")
	}
	if orgIDStr == "" {
		slog.Warn("ws: upgrade missing orgId")
		return c.JSON(http.StatusUnprocessableEntity, map[string]string{
			"error":   "validation_error",
			"message": "orgId query required (UUID)",
		})
	}
	orgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		slog.Warn("ws: upgrade invalid orgId", "orgId", orgIDStr, "error", err)
		return c.JSON(http.StatusUnprocessableEntity, map[string]string{
			"error":   "validation_error",
			"message": "orgId must be UUID",
		})
	}

	conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		slog.Error("ws: upgrade failed", "orgId", orgID.String(), "error", err)
		return err
	}

	client := NewClient(h.hub, orgID, conn)
	h.hub.Register(orgID, client)
	// Ensure cleanup: Unregister closes send channel, WritePump will exit on close.
	defer h.hub.Unregister(orgID, client)

	slog.Info("ws: client connected", "orgId", orgID.String(), "remote", c.Request().RemoteAddr)

	// Start writer in goroutine, reader blocks.
	go client.WritePump()
	client.ReadPump()

	slog.Info("ws: client disconnected", "orgId", orgID.String())
	return nil
}
