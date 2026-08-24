package ws

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/labstack/echo/v4"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// For MVP allow all origins; production should check AllowedOrigins
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Handler handles GET /ws?orgId=...
type Handler struct {
	hub *Hub
}

// NewHandler creates WS handler.
func NewHandler(hub *Hub) *Handler {
	return &Handler{hub: hub}
}

// RegisterRoutes mounts GET /ws under group (e.g., /api/v1)
func (h *Handler) RegisterRoutes(g *echo.Group) {
	g.GET("/ws", h.Upgrade)
}

// Upgrade upgrades to WS and subscribes to orgId broadcasts.
func (h *Handler) Upgrade(c echo.Context) error {
	orgIDStr := c.QueryParam("orgId")
	if orgIDStr == "" {
		// also allow orgID lowercase?
		orgIDStr = c.QueryParam("orgID")
	}
	if orgIDStr == "" {
		return c.JSON(http.StatusUnprocessableEntity, map[string]string{"error": "validation_error", "message": "orgId query required"})
	}
	orgID, err := uuid.Parse(orgIDStr)
	if err != nil {
		return c.JSON(http.StatusUnprocessableEntity, map[string]string{"error": "validation_error", "message": "orgId must be UUID"})
	}
	conn, err := upgrader.Upgrade(c.Response(), c.Request(), nil)
	if err != nil {
		return err
	}
	client := NewClient(h.hub, orgIDStr, conn)
	h.hub.Register(orgID, client)
	// Ensure cleanup
	defer h.hub.Unregister(orgID, client)
	go client.WritePump()
	client.ReadPump()
	return nil
}
