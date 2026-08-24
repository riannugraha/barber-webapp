package middleware

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// RequireRole returns middleware that allows only the given roles.
// It must be used AFTER JWTMiddleware — it reads *Claims from context.
// If no user is authenticated -> 401. If role not allowed -> 403.
// Example: RequireRole("OWNER", "STAFF") -> CUSTOMER POST /services gets 403.
func RequireRole(roles ...string) echo.MiddlewareFunc {
	allowed := make(map[string]struct{}, len(roles))
	for _, r := range roles {
		allowed[r] = struct{}{}
	}

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			claims := GetClaims(c)
			if claims == nil {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error":   "unauthorized",
					"message": "authentication required",
				})
			}
			if _, ok := allowed[claims.Role]; !ok {
				return c.JSON(http.StatusForbidden, map[string]string{
					"error":   "forbidden",
					"message": "insufficient role: requires " + joinRoles(roles),
				})
			}
			return next(c)
		}
	}
}

func joinRoles(roles []string) string {
	if len(roles) == 0 {
		return ""
	}
	out := roles[0]
	for _, r := range roles[1:] {
		out += "/" + r
	}
	return out
}
