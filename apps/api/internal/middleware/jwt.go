package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

// Context keys for authenticated user
const (
	ContextKeyUser     = "user"
	ContextKeyUserID   = "userID"
	ContextKeyUserRole = "userRole"
	ContextKeyUserEmail = "userEmail"
	ContextKeyOrgID    = "orgID"
)

// Claims is the JWT access token payload (15m).
type Claims struct {
	UserID string  `json:"user_id"`
	Email  string  `json:"email"`
	Role   string  `json:"role"`
	OrgID  *string `json:"org_id,omitempty"`
	jwt.RegisteredClaims
}

// JWTMiddleware validates Authorization: Bearer <access 15m>.
// On success it stores *Claims in context under ContextKeyUser and returns Next.
// On failure it returns 401 JSON — Next middleware (frontend) will redirect to /login.
func JWTMiddleware(secret string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if secret == "" {
				return c.JSON(http.StatusInternalServerError, map[string]string{
					"error":   "server_misconfigured",
					"message": "JWT_SECRET not configured",
				})
			}
			auth := c.Request().Header.Get(echo.HeaderAuthorization)
			if auth == "" {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error":   "unauthorized",
					"message": "missing Authorization header",
				})
			}
			parts := strings.SplitN(auth, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error":   "unauthorized",
					"message": "Authorization header must be Bearer <token>",
				})
			}
			tokenString := strings.TrimSpace(parts[1])
			if tokenString == "" {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error":   "unauthorized",
					"message": "empty bearer token",
				})
			}

			claims := &Claims{}
			token, err := jwt.ParseWithClaims(tokenString, claims, func(t *jwt.Token) (interface{}, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
				}
				return []byte(secret), nil
			}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Name}))

			if err != nil || !token.Valid {
				msg := "invalid or expired token"
				if err != nil {
					msg = err.Error()
					// hide internal details for expired token but keep message useful
					if strings.Contains(strings.ToLower(msg), "expired") {
						msg = "token expired"
					}
				}
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error":   "unauthorized",
					"message": msg,
				})
			}

			if claims.UserID == "" || claims.Role == "" {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error":   "unauthorized",
					"message": "invalid token claims",
				})
			}

			// Store claims for downstream handlers / RequireRole
			c.Set(ContextKeyUser, claims)
			c.Set(ContextKeyUserID, claims.UserID)
			c.Set(ContextKeyUserRole, claims.Role)
			c.Set(ContextKeyUserEmail, claims.Email)
			if claims.OrgID != nil {
				c.Set(ContextKeyOrgID, *claims.OrgID)
			}
			return next(c)
		}
	}
}

// GetClaims retrieves *Claims from context, or nil if not authenticated.
// Helper for handlers that need user info without re-parsing.
func GetClaims(c echo.Context) *Claims {
	v := c.Get(ContextKeyUser)
	if v == nil {
		return nil
	}
	if cl, ok := v.(*Claims); ok {
		return cl
	}
	return nil
}

// GetUserID returns the authenticated user ID or empty string.
func GetUserID(c echo.Context) string {
	if cl := GetClaims(c); cl != nil {
		return cl.UserID
	}
	if v := c.Get(ContextKeyUserID); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// GetUserRole returns the authenticated user role or empty string.
func GetUserRole(c echo.Context) string {
	if cl := GetClaims(c); cl != nil {
		return cl.Role
	}
	if v := c.Get(ContextKeyUserRole); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
