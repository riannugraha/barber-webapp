package health

import (
	"context"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
)

// Version is the API version reported by /health — keep in sync with openapi.yaml info.version.
const Version = "1.0.0"

// Handler exposes GET /health for Koyeb health check and Vercel Cron ping.
// It pings pgxpool (pooler 6543 transaction mode) with 2s timeout.
type Handler struct {
	pool    *pgxpool.Pool
	version string
}

// New creates a health handler. pool may be nil (reports db:down).
func New(pool *pgxpool.Pool) *Handler {
	return &Handler{pool: pool, version: Version}
}

// Handle is echo.HandlerFunc for GET /health and GET /api/v1/health.
// Always returns 200 with {"status":"ok","db":"up|down","version":"1.0.0"}.
// Koyeb expects 200 even when db is down — status field indicates readiness.
func (h *Handler) Handle(c echo.Context) error {
	dbStatus := "up"
	if h.pool == nil {
		dbStatus = "down"
		return c.JSON(http.StatusOK, map[string]string{
			"status":  "ok",
			"db":      dbStatus,
			"version": h.version,
		})
	}
	ctx, cancel := context.WithTimeout(c.Request().Context(), 2*time.Second)
	defer cancel()
	if err := h.pool.Ping(ctx); err != nil {
		dbStatus = "down"
		return c.JSON(http.StatusOK, map[string]string{
			"status":  "ok",
			"db":      dbStatus,
			"version": h.version,
			"error":   err.Error(),
		})
	}
	return c.JSON(http.StatusOK, map[string]string{
		"status":  "ok",
		"db":      dbStatus,
		"version": h.version,
	})
}

// HealthHandler is a convenience for wiring without struct — used by main.go.
func HealthHandler(pool *pgxpool.Pool) echo.HandlerFunc {
	h := New(pool)
	return h.Handle
}

// RegisterRoutes mounts /health on both root and v1 groups.
// Call with e.GET("/health", ...) and v1.GET("/health", ...).
func (h *Handler) RegisterRoutes(e *echo.Echo, v1 *echo.Group) {
	e.GET("/health", h.Handle)
	if v1 != nil {
		v1.GET("/health", h.Handle)
	}
}
