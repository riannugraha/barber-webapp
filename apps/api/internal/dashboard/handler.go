package dashboard

import (
	"net/http"
	"strings"
	"time"

	"flowbook/api/internal/middleware"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// Handler exposes GET /dashboard — 5-row aggregates with DATE_TRUNC + GROUP BY in DB, index start_at, not JS.
// OWNER full, STAFF scoped miliknya, tabular-nums on frontend.
type Handler struct {
	svc       *Service
	validator *validator.Validate
}

// NewHandler creates dashboard handler.
func NewHandler(svc *Service) *Handler {
	v := validator.New(validator.WithRequiredStructEnabled())
	return &Handler{svc: svc, validator: v}
}

// RegisterRoutes mounts GET /dashboard under group with JWT + RequireRole(OWNER,STAFF).
// Caller should pass jwtSecret; if empty, handler will still check claims and return 401.
func (h *Handler) RegisterRoutes(g *echo.Group, jwtSecret string) {
	if jwtSecret != "" {
		g.GET("/dashboard", h.GetDashboard, middleware.JWTMiddleware(jwtSecret), middleware.RequireRole("OWNER", "STAFF"))
	} else {
		g.GET("/dashboard", h.GetDashboard)
	}
}

// query DTO for validation — all query params are strings from URL.
type dashboardQuery struct {
	From        string `validate:"omitempty"`
	To          string `validate:"omitempty"`
	Granularity string `validate:"omitempty,oneof=day week month"`
	TZ          string `validate:"omitempty"`
}

// error response shapes — Zod-compatible 422.
type errorResponse struct {
	Error   string       `json:"error"`
	Message string       `json:"message,omitempty"`
	Details []fieldError `json:"details,omitempty"`
}

type fieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// GetDashboard handles GET /dashboard?from&to&granularity&tz
// Returns 5-row: kpi {revenue, bookings, occupancy, avgTicket, delta}, area 10 titik Nov2025->Agu2026, pie Classic Cut 35%, bar Andi 90/Bayu70/Sari20, heatmap 7x15, topCustomers 15 Siti 18x, recent 10, insights {busiestMonth Des 2025, cancelRate 7.2%, utilization}.
func (h *Handler) GetDashboard(c echo.Context) error {
	// Parse query params
	q := dashboardQuery{
		From:        c.QueryParam("from"),
		To:          c.QueryParam("to"),
		Granularity: c.QueryParam("granularity"),
		TZ:          c.QueryParam("tz"),
	}
	if q.Granularity == "" {
		q.Granularity = "month"
	}
	if q.TZ == "" {
		q.TZ = "Asia/Jakarta"
	}
	if err := h.validator.Struct(q); err != nil {
		return c.JSON(http.StatusUnprocessableEntity, errorResponse{
			Error:   "validation_error",
			Message: "Validation failed",
			Details: []fieldError{{Field: "granularity", Message: "must be one of day, week, month"}},
		})
	}
	// Validate TZ is IANA
	if _, err := time.LoadLocation(q.TZ); err != nil {
		return c.JSON(http.StatusUnprocessableEntity, errorResponse{
			Error:   "validation_error",
			Message: "Validation failed",
			Details: []fieldError{{Field: "tz", Message: "must be a valid IANA timezone"}},
		})
	}

	// Parse from/to as date YYYY-MM-DD or RFC3339. Store as UTC.
	var fromPtr, toPtr *time.Time
	if q.From != "" {
		t, err := parseDateParam(q.From, q.TZ)
		if err != nil {
			return c.JSON(http.StatusUnprocessableEntity, errorResponse{
				Error:   "validation_error",
				Message: "Validation failed",
				Details: []fieldError{{Field: "from", Message: "must be YYYY-MM-DD or RFC3339"}},
			})
		}
		fromPtr = &t
	}
	if q.To != "" {
		t, err := parseDateParam(q.To, q.TZ)
		if err != nil {
			return c.JSON(http.StatusUnprocessableEntity, errorResponse{
				Error:   "validation_error",
				Message: "Validation failed",
				Details: []fieldError{{Field: "to", Message: "must be YYYY-MM-DD or RFC3339"}},
			})
		}
		// If to was date-only, set to end of day in TZ
		if len(q.To) == 10 {
			// parseDateParam already returns 00:00 in TZ converted to UTC, so add 24h -1ns
			endOfDay := t.Add(24*time.Hour - time.Nanosecond)
			toPtr = &endOfDay
		} else {
			toPtr = &t
		}
	}

	// Auth — get claims
	claims := middleware.GetClaims(c)
	if claims == nil {
		return c.JSON(http.StatusUnauthorized, errorResponse{
			Error:   "unauthorized",
			Message: "authentication required",
		})
	}
	// OrgID from claims
	var orgID uuid.UUID
	if claims.OrgID != nil && *claims.OrgID != "" {
		parsed, err := uuid.Parse(*claims.OrgID)
		if err != nil {
			return c.JSON(http.StatusUnprocessableEntity, errorResponse{
				Error:   "validation_error",
				Message: "Invalid organizationId from token",
				Details: []fieldError{{Field: "organizationId", Message: "must be a valid UUID"}},
			})
		}
		orgID = parsed
	} else {
		// Fallback to query param organizationId for testing (optional)
		if qOrg := c.QueryParam("organizationId"); qOrg != "" {
			parsed, err := uuid.Parse(qOrg)
			if err != nil {
				return c.JSON(http.StatusUnprocessableEntity, errorResponse{
					Error:   "validation_error",
					Message: "Invalid organizationId",
					Details: []fieldError{{Field: "organizationId", Message: "must be a valid UUID"}},
				})
			}
			orgID = parsed
		} else {
			return c.JSON(http.StatusUnprocessableEntity, errorResponse{
				Error:   "validation_error",
				Message: "organizationId required from token",
				Details: []fieldError{{Field: "organizationId", Message: "missing org_id in token"}},
			})
		}
	}
	// UserID for staff scoping
	var userID uuid.UUID
	if claims.UserID != "" {
		if parsed, err := uuid.Parse(claims.UserID); err == nil {
			userID = parsed
		}
	}
	role := claims.Role

	// Call service
	params := GetDashboardParams{
		OrgID:       orgID,
		UserID:      userID,
		Role:        role,
		From:        fromPtr,
		To:          toPtr,
		Granularity: strings.ToLower(q.Granularity),
		TZ:          q.TZ,
	}
	resp, err := h.svc.GetDashboard(c.Request().Context(), params)
	if err != nil {
		// Map known errors
		if err == ErrInvalidRange || err == ErrInvalidGran || err == ErrInvalidTimezone || err == ErrOrgRequired {
			return c.JSON(http.StatusUnprocessableEntity, errorResponse{
				Error:   "validation_error",
				Message: err.Error(),
			})
		}
		if err == ErrForbidden {
			return c.JSON(http.StatusForbidden, errorResponse{
				Error:   "forbidden",
				Message: "insufficient role",
			})
		}
		if strings.Contains(err.Error(), "invalid timezone") || strings.Contains(err.Error(), "invalid granularity") {
			return c.JSON(http.StatusUnprocessableEntity, errorResponse{
				Error:   "validation_error",
				Message: err.Error(),
			})
		}
		// Log internal via echo logger? For now return 500
		return c.JSON(http.StatusInternalServerError, errorResponse{
			Error:   "internal_error",
			Message: err.Error(),
		})
	}

	return c.JSON(http.StatusOK, resp)
}

// parseDateParam parses YYYY-MM-DD or RFC3339. If YYYY-MM-DD, interprets as date in tz at 00:00.
// Returns time in UTC.
func parseDateParam(s, tz string) (time.Time, error) {
	// Try RFC3339 first
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t.UTC(), nil
	}
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.UTC(), nil
	}
	// Try date only YYYY-MM-DD
	if len(s) == 10 {
		loc, err := time.LoadLocation(tz)
		if err != nil {
			loc = time.UTC
		}
		t, err := time.ParseInLocation("2006-01-02", s, loc)
		if err != nil {
			return time.Time{}, err
		}
		return t.UTC(), nil
	}
	// Fallback try date in UTC
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t.UTC(), nil
	}
	return time.Time{}, &time.ParseError{Layout: "2006-01-02 or RFC3339", Value: s, LayoutElem: "", ValueElem: s, Message: "invalid date"}
}
