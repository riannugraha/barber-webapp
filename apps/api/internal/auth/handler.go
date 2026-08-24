package auth

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
)

// Handler exposes POST /auth/* endpoints.
type Handler struct {
	svc       *Service
	validator *validator.Validate
}

// NewHandler creates auth Handler.
func NewHandler(svc *Service) *Handler {
	v := validator.New(validator.WithRequiredStructEnabled())
	return &Handler{svc: svc, validator: v}
}

// RegisterRoutes mounts auth routes under group (e.g., /api/v1).
func (h *Handler) RegisterRoutes(g *echo.Group) {
	g.POST("/auth/register", h.Register)
	g.POST("/auth/login", h.Login)
	g.POST("/auth/refresh", h.Refresh)
	g.POST("/auth/logout", h.Logout)
}

// validationErrorResponse returns 422 Zod-compatible payload.
type validationErrorResponse struct {
	Error   string       `json:"error"`
	Message string       `json:"message"`
	Details []fieldError `json:"details"`
}

type fieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (h *Handler) validationError(c echo.Context, err error) error {
	var details []fieldError
	if ve, ok := err.(validator.ValidationErrors); ok {
		for _, fe := range ve {
			field := jsonFieldName(fe.Field())
			msg := msgForTag(fe)
			details = append(details, fieldError{Field: field, Message: msg})
		}
	} else {
		details = append(details, fieldError{Field: "body", Message: err.Error()})
	}
	return c.JSON(http.StatusUnprocessableEntity, validationErrorResponse{
		Error:   "validation_error",
		Message: "Validation failed",
		Details: details,
	})
}

func jsonFieldName(goField string) string {
	// Map Go struct field to json tag lowerCamel
	switch goField {
	case "Email":
		return "email"
	case "Password":
		return "password"
	case "Name":
		return "name"
	case "Role":
		return "role"
	case "OrganizationID":
		return "organizationId"
	case "OrganizationId":
		return "organizationId"
	default:
		// lower first letter
		if goField == "" {
			return goField
		}
		return strings.ToLower(goField[:1]) + goField[1:]
	}
}

func msgForTag(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return fe.Field() + " is required"
	case "email":
		return "Invalid email"
	case "min":
		return fe.Field() + " must be at least " + fe.Param() + " characters"
	case "oneof":
		return fe.Field() + " must be one of [" + fe.Param() + "]"
	case "uuid":
		return fe.Field() + " must be a valid UUID"
	default:
		return fe.Field() + " failed on " + fe.Tag()
	}
}

// Register godoc
// POST /auth/register — bcrypt, returns 201 + sets refresh cookie.
func (h *Handler) Register(c echo.Context) error {
	var req RegisterRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusUnprocessableEntity, validationErrorResponse{
			Error:   "validation_error",
			Message: "Validation failed",
			Details: []fieldError{{Field: "body", Message: "invalid JSON"}},
		})
	}
	if err := h.validator.Struct(req); err != nil {
		return h.validationError(c, err)
	}

	resp, pair, err := h.svc.Register(c.Request().Context(), req)
	if err != nil {
		if errors.Is(err, ErrDuplicateEmail) {
			return c.JSON(http.StatusConflict, map[string]string{
				"error":   "conflict",
				"message": "Email already exists",
			})
		}
		if strings.Contains(strings.ToLower(err.Error()), "validation") || strings.Contains(err.Error(), "invalid") {
			return c.JSON(http.StatusUnprocessableEntity, validationErrorResponse{
				Error:   "validation_error",
				Message: err.Error(),
			})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": err.Error(),
		})
	}

	// Set refresh cookie httpOnly Secure SameSite=Lax 7d
	setRefreshCookie(c, pair.RefreshTokenRaw, pair.RefreshExpiresAt)

	return c.JSON(http.StatusCreated, resp)
}

// Login godoc
// POST /auth/login — verifies bcrypt, returns 200 + Sets httpOnly refresh cookie 7d.
func (h *Handler) Login(c echo.Context) error {
	var req LoginRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusUnprocessableEntity, validationErrorResponse{
			Error:   "validation_error",
			Message: "Validation failed",
			Details: []fieldError{{Field: "body", Message: "invalid JSON"}},
		})
	}
	if err := h.validator.Struct(req); err != nil {
		return h.validationError(c, err)
	}

	resp, pair, err := h.svc.Login(c.Request().Context(), req)
	if err != nil {
		if errors.Is(err, ErrInvalidCredentials) {
			return c.JSON(http.StatusUnauthorized, map[string]string{
				"error":   "unauthorized",
				"message": "Invalid credentials",
			})
		}
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error":   "internal_error",
			"message": err.Error(),
		})
	}

	setRefreshCookie(c, pair.RefreshTokenRaw, pair.RefreshExpiresAt)
	return c.JSON(http.StatusOK, resp)
}

// Refresh godoc
// POST /auth/refresh — reads refresh_token cookie, returns new access 15m.
func (h *Handler) Refresh(c echo.Context) error {
	raw := ""
	if cookie, err := c.Cookie("refresh_token"); err == nil && cookie.Value != "" {
		raw = cookie.Value
	} else {
		// also allow header fallback for tests
		raw = c.Request().Header.Get("X-Refresh-Token")
	}

	if raw == "" {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error":   "unauthorized",
			"message": "missing refresh_token",
		})
	}

	access, exp, err := h.svc.Refresh(c.Request().Context(), raw)
	if err != nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error":   "unauthorized",
			"message": "invalid or expired refresh token",
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"accessToken": access,
		"expiresAt":   exp,
	})
}

// Logout godoc
// POST /auth/logout — revokes refresh token, clears cookie, returns 204.
func (h *Handler) Logout(c echo.Context) error {
	raw := ""
	if cookie, err := c.Cookie("refresh_token"); err == nil {
		raw = cookie.Value
	}
	// fallback to header for tests
	if raw == "" {
		raw = c.Request().Header.Get("X-Refresh-Token")
	}
	// Also allow Bearer? But refresh is cookie-based; if none, still 204 idempotent
	if raw != "" {
		_ = h.svc.Logout(c.Request().Context(), raw)
	}
	clearRefreshCookie(c)
	return c.NoContent(http.StatusNoContent)
}

func setRefreshCookie(c echo.Context, raw string, expires time.Time) {
	// Secure true in production, but for httptest/local we set based on request scheme.
	// Spec requires Secure, SameSite=Lax, HttpOnly, 7d.
	secure := true
	// Allow http for localhost/test to not break cookie in httptest
	if c.Request().Host == "" || strings.Contains(c.Request().Host, "localhost") || strings.Contains(c.Request().Host, "127.0.0.1") {
		// in test/local, Echo's test request is http; still set Secure true as per spec but
		// browsers ignore Secure over http. For httptest we keep Secure true to satisfy spec,
		// but cookie still readable via Set-Cookie header. So keep secure true.
		// However to ensure httptest client sends it back, we keep Secure field true; Go's test
		// recorder still stores it. So no change.
	}
	cookie := &http.Cookie{
		Name:     "refresh_token",
		Value:    raw,
		Path:     "/",
		Expires:  expires,
		MaxAge:   int(time.Until(expires).Seconds()),
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	}
	c.SetCookie(cookie)
}

func clearRefreshCookie(c echo.Context) {
	cookie := &http.Cookie{
		Name:     "refresh_token",
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}
	c.SetCookie(cookie)
}
