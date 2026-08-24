package middleware

import (
	echoMw "github.com/labstack/echo/v4/middleware"
	"github.com/labstack/echo/v4"
)

// RequestID returns Echo RequestID middleware.
// It generates X-Request-ID if not present and echoes it back.
func RequestID() echo.MiddlewareFunc {
	return echoMw.RequestID()
}

// RequestLogger returns a structured logger middleware using echo's Logger.
// For slog JSON we use the global slog in main.go; this is a lightweight wrapper.
func RequestLogger() echo.MiddlewareFunc {
	return echoMw.LoggerWithConfig(echoMw.LoggerConfig{
		Format: `${time_rfc3339} ${id} ${method} ${uri} ${status} ${latency_human} ${error}` + "\n",
	})
}
