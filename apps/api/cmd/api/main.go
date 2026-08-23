package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// Config holds env-driven configuration (Koyeb/Supabase).
type Config struct {
	Port         string
	DatabaseURL  string
	JWTSecret    string
	AppEnv       string
	TestSecret   string
	AllowedOrigins []string
}

func loadConfig() Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	return Config{
		Port:        port,
		DatabaseURL: os.Getenv("DATABASE_URL"),
		JWTSecret:   os.Getenv("JWT_SECRET"),
		AppEnv:      os.Getenv("APP_ENV"),
		TestSecret:  os.Getenv("TEST_SECRET"),
		AllowedOrigins: []string{
			"https://flowbook-xxx.vercel.app",
			"http://localhost:3000",
		},
	}
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg := loadConfig()

	// pgxpool to Supabase pooler 6543 transaction mode — never database/sql generic for hot path
	var pool *pgxpool.Pool
	if cfg.DatabaseURL != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		p, err := pgxpool.New(ctx, cfg.DatabaseURL)
		if err != nil {
			slog.Error("failed to create pgxpool", "error", err)
		} else {
			pool = p
			defer pool.Close()
			if err := pool.Ping(ctx); err != nil {
				slog.Error("db ping failed", "error", err)
			} else {
				slog.Info("db connected", "pooler", "6543")
			}
		}
	} else {
		slog.Warn("DATABASE_URL not set — running without DB (health will report down)")
	}

	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	// Global middleware
	e.Use(middleware.RequestID())
	e.Use(middleware.Recover())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: cfg.AllowedOrigins,
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization, "x-test-secret"},
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions, http.MethodPatch},
		AllowCredentials: true,
	}))
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Format: `${time_rfc3339} ${method} ${uri} ${status} ${latency_human}` + "\n",
	}))

	// Health — Koyeb expects GET /health at root
	e.GET("/health", healthHandler(pool))
	e.GET("/api/v1/health", healthHandler(pool))
	e.GET("/", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ok", "service": "flowbook-api"})
	})

	// OpenAPI spec — serve file for orval
	e.GET("/openapi.yaml", func(c echo.Context) error {
		return c.File("openapi.yaml")
	})
	e.GET("/api/v1/openapi.yaml", func(c echo.Context) error {
		return c.File("openapi.yaml")
	})

	// API v1 group (stub — full handlers in T02+)
	v1 := e.Group("/api/v1")
	v1.GET("/health", healthHandler(pool))
	v1.GET("/openapi.yaml", func(c echo.Context) error {
		return c.File("openapi.yaml")
	})

	// Test-only helpers — only when APP_ENV=test + x-test-secret
	if cfg.AppEnv == "test" {
		v1.POST("/test/reset", testResetHandler(pool, cfg.TestSecret))
		v1.POST("/test/seed-full", testSeedFullHandler(pool, cfg.TestSecret))
		slog.Info("test helpers enabled", "env", "test")
	}

	slog.Info("starting server", "port", cfg.Port, "env", cfg.AppEnv)
	if err := e.Start(":" + cfg.Port); err != nil && err != http.ErrServerClosed {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}
}

func healthHandler(pool *pgxpool.Pool) echo.HandlerFunc {
	return func(c echo.Context) error {
		dbStatus := "up"
		if pool == nil {
			dbStatus = "down"
		} else {
			ctx, cancel := context.WithTimeout(c.Request().Context(), 2*time.Second)
			defer cancel()
			if err := pool.Ping(ctx); err != nil {
				dbStatus = "down"
				return c.JSON(http.StatusOK, map[string]string{"status": "ok", "db": dbStatus, "error": err.Error()})
			}
		}
		return c.JSON(http.StatusOK, map[string]string{"status": "ok", "db": dbStatus})
	}
}

func testOnlyGuard(c echo.Context, expected string) bool {
	if expected == "" {
		return false
	}
	return c.Request().Header.Get("x-test-secret") == expected
}

func testResetHandler(pool *pgxpool.Pool, secret string) echo.HandlerFunc {
	return func(c echo.Context) error {
		if !testOnlyGuard(c, secret) {
			return echo.NewHTTPError(http.StatusUnauthorized, map[string]string{"error": "unauthorized", "message": "invalid x-test-secret or APP_ENV!=test"})
		}
		if pool == nil {
			return echo.NewHTTPError(http.StatusServiceUnavailable, map[string]string{"error": "db_unavailable"})
		}
		ctx := c.Request().Context()
		// Opsi A: TRUNCATE + seedMinimal — 150ms
		_, err := pool.Exec(ctx, `TRUNCATE bookings, payments, customers RESTART IDENTITY CASCADE`)
		if err != nil {
			slog.Error("test reset truncate failed", "error", err)
			return echo.NewHTTPError(http.StatusInternalServerError, map[string]string{"error": "truncate_failed", "message": err.Error()})
		}
		// seedMinimal is stubbed — full seed in cmd/seed
		slog.Info("test reset done")
		return c.NoContent(http.StatusNoContent)
	}
}

func testSeedFullHandler(pool *pgxpool.Pool, secret string) echo.HandlerFunc {
	return func(c echo.Context) error {
		if !testOnlyGuard(c, secret) {
			return echo.NewHTTPError(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		}
		if pool == nil {
			return echo.NewHTTPError(http.StatusServiceUnavailable, map[string]string{"error": "db_unavailable"})
		}
		slog.Info("test seed-full requested (implement in cmd/seed)")
		return c.NoContent(http.StatusNoContent)
	}
}
