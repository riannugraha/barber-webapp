package middleware

import (
	"net/http"

	"github.com/labstack/echo/v4"
	echoMw "github.com/labstack/echo/v4/middleware"
)

// CORS returns Echo CORS middleware with FlowBook allowed origins.
// AllowedOrigins: https://flowbook-xxx.vercel.app + http://localhost:3000
// Must allow credentials (cookies for refresh) and required headers.
func CORS(allowedOrigins []string) echo.MiddlewareFunc {
	if len(allowedOrigins) == 0 {
		allowedOrigins = []string{
			"https://flowbook-xxx.vercel.app",
			"http://localhost:3000",
		}
	}
	return echoMw.CORSWithConfig(echoMw.CORSConfig{
		AllowOrigins:     allowedOrigins,
		AllowHeaders:     []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization, "x-test-secret"},
		AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions, http.MethodPatch},
		AllowCredentials: true,
	})
}
