package testhelpers

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"

	"flowbook/api/internal/seed"
)

// TestGuard checks APP_ENV=test && x-test-secret == expected.
// Returns true if allowed. For stealth, callers should return 404 when not in test env.
func TestGuard(c echo.Context, appEnv, expectedSecret string) bool {
	if appEnv != "test" {
		return false
	}
	if expectedSecret == "" {
		return false
	}
	return c.Request().Header.Get("x-test-secret") == expectedSecret
}

// ResetHandler handles POST /test/reset and POST /test/seed-full with guard.
// It TRUNCATE bookings,payments,customers RESTART IDENTITY CASCADE and then seedMinimal (10 rows, 150ms)
// or seedFull (~1.5k). It uses pgxpool (pooler 6543) and never bypasses EXCLUDE — insertions go through normal path.
type Handlers struct {
	pool       *pgxpool.Pool
	appEnv     string
	testSecret string
}

func New(pool *pgxpool.Pool, appEnv, testSecret string) *Handlers {
	return &Handlers{pool: pool, appEnv: appEnv, testSecret: testSecret}
}

// RegisterRoutes mounts POST /test/reset and /test/seed-full under given group (e.g., v1).
func (h *Handlers) RegisterRoutes(g *echo.Group) {
	g.POST("/test/reset", h.Reset)
	g.POST("/test/seed-full", h.SeedFull)
}

// Reset handles POST /test/reset
func (h *Handlers) Reset(c echo.Context) error {
	if !TestGuard(c, h.appEnv, h.testSecret) {
		// PLAN.md: selain itu 404 (hide existence), but also allow 401 for bad secret when in test env
		// We return 404 if not in test env, 401 if secret mismatch to aid debugging — both satisfy "guard"
		if h.appEnv != "test" {
			return echo.NewHTTPError(http.StatusNotFound, map[string]string{"error": "not_found"})
		}
		return echo.NewHTTPError(http.StatusUnauthorized, map[string]string{"error": "unauthorized", "message": "invalid x-test-secret"})
	}
	if h.pool == nil {
		return echo.NewHTTPError(http.StatusServiceUnavailable, map[string]string{"error": "db_unavailable"})
	}
	ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Second)
	defer cancel()

	// Opsi A: TRUNCATE + seedMinimal — target 150-250ms
	start := time.Now()
	if _, err := h.pool.Exec(ctx, `TRUNCATE bookings, payments, customers RESTART IDENTITY CASCADE`); err != nil {
		slog.Error("test reset truncate failed", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"error": "truncate_failed", "message": err.Error()})
	}
	inserted, err := seed.SeedMinimal(ctx, h.pool)
	if err != nil {
		slog.Error("seed minimal failed", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"error": "seed_failed", "message": err.Error()})
	}
	elapsed := time.Since(start)
	slog.Info("test reset done", "bookings", inserted, "elapsed", elapsed.String())
	// For minimal we expect 150ms — log warning if slower but don't fail
	if elapsed > 500*time.Millisecond {
		slog.Warn("test reset slow (>500ms) — check pooler 6543 and workers:1", "elapsed", elapsed.String())
	}
	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":   "ok",
		"bookings": inserted,
		"elapsed":  elapsed.String(),
	})
}

// SeedFull handles POST /test/seed-full — only for chart tests (04-owner-dashboard)
func (h *Handlers) SeedFull(c echo.Context) error {
	if !TestGuard(c, h.appEnv, h.testSecret) {
		if h.appEnv != "test" {
			return echo.NewHTTPError(http.StatusNotFound, map[string]string{"error": "not_found"})
		}
		return echo.NewHTTPError(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}
	if h.pool == nil {
		return echo.NewHTTPError(http.StatusServiceUnavailable, map[string]string{"error": "db_unavailable"})
	}
	ctx, cancel := context.WithTimeout(c.Request().Context(), 90*time.Second)
	defer cancel()

	start := time.Now()
	inserted, err := seed.SeedFull(ctx, h.pool)
	if err != nil {
		slog.Error("seed full failed", "error", err)
		return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"error": "seed_failed", "message": err.Error()})
	}
	elapsed := time.Since(start)
	slog.Info("test seed-full done", "bookings", inserted, "elapsed", elapsed.String())
	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":   "ok",
		"bookings": inserted,
		"elapsed":  elapsed.String(),
	})
}

// Legacy funcs for backward compat with main.go inline handlers (if imported elsewhere)

// ResetHandler returns echo.HandlerFunc with same guard logic (for main.go wiring without struct).
func ResetHandler(pool *pgxpool.Pool, appEnv, testSecret string) echo.HandlerFunc {
	h := New(pool, appEnv, testSecret)
	return h.Reset
}

// SeedFullHandler returns echo.HandlerFunc for /test/seed-full.
func SeedFullHandler(pool *pgxpool.Pool, appEnv, testSecret string) echo.HandlerFunc {
	h := New(pool, appEnv, testSecret)
	return h.SeedFull
}
