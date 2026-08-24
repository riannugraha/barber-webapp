package availability

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// Handler exposes GET /availability/slots
type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// RegisterRoutes mounts handler under group. e.g., v1.GET("/availability/slots", h.GetSlots)
func (h *Handler) RegisterRoutes(g *echo.Group) {
	g.GET("/availability/slots", h.GetSlots)
}

// GetSlots godoc
// @Summary Get available slots for a date (timezone-aware, cached 30s)
// @Param serviceId query string true "Service determines duration+buffer"
// @Param staffId query string false "Optional — if omitted returns all eligible staff slots (Any available union)"
// @Param date query string true "Date YYYY-MM-DD in requested timezone"
// @Param tz query string false "IANA timezone — default Asia/Jakarta"
// @Success 200 {object} SlotsResponse
// @Failure 422 {object} ErrorResponse
func (h *Handler) GetSlots(c echo.Context) error {
	serviceID := c.QueryParam("serviceId")
	staffID := c.QueryParam("staffId")
	date := c.QueryParam("date")
	tz := c.QueryParam("tz")
	if tz == "" {
		tz = "Asia/Jakarta"
	}

	// Validation 422 Zod-compatible
	if serviceID == "" {
		return c.JSON(http.StatusUnprocessableEntity, ErrorResponse{
			Error:   "validation_error",
			Message: "Validation failed",
			Details: []FieldError{{Field: "serviceId", Message: "serviceId is required"}},
		})
	}
	if _, err := uuid.Parse(serviceID); err != nil {
		return c.JSON(http.StatusUnprocessableEntity, ErrorResponse{
			Error:   "validation_error",
			Message: "Validation failed",
			Details: []FieldError{{Field: "serviceId", Message: "must be a valid UUID"}},
		})
	}
	if date == "" {
		return c.JSON(http.StatusUnprocessableEntity, ErrorResponse{
			Error:   "validation_error",
			Message: "Validation failed",
			Details: []FieldError{{Field: "date", Message: "date is required (YYYY-MM-DD)"}},
		})
	}
	if _, err := time.Parse("2006-01-02", date); err != nil {
		return c.JSON(http.StatusUnprocessableEntity, ErrorResponse{
			Error:   "validation_error",
			Message: "Validation failed",
			Details: []FieldError{{Field: "date", Message: "must be YYYY-MM-DD"}},
		})
	}
	if staffID != "" {
		if _, err := uuid.Parse(staffID); err != nil {
			return c.JSON(http.StatusUnprocessableEntity, ErrorResponse{
				Error:   "validation_error",
				Message: "Validation failed",
				Details: []FieldError{{Field: "staffId", Message: "must be a valid UUID"}},
			})
		}
	}
	if tz != "" {
		if _, err := time.LoadLocation(tz); err != nil {
			return c.JSON(http.StatusUnprocessableEntity, ErrorResponse{
				Error:   "validation_error",
				Message: "Validation failed",
				Details: []FieldError{{Field: "tz", Message: "must be a valid IANA timezone"}},
			})
		}
	}

	slots, resolvedTZ, err := h.svc.GetSlots(c.Request().Context(), serviceID, staffID, date, tz)
	if err != nil {
		// Map known errors
		switch {
		case isInvalidError(err):
			return c.JSON(http.StatusUnprocessableEntity, ErrorResponse{
				Error:   "validation_error",
				Message: err.Error(),
			})
		case isNotFoundError(err):
			return c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: err.Error(),
			})
		default:
			// Log is done by middleware; return 500
			return c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error:   "internal_error",
				Message: err.Error(),
			})
		}
	}

	resp := SlotsResponse{
		Date:  date,
		TZ:    resolvedTZ,
		Slots: slots,
	}
	return c.JSON(http.StatusOK, resp)
}

// SlotsResponse matches openapi.yaml GET /availability/slots 200
type SlotsResponse struct {
	Date  string `json:"date"`
	TZ    string `json:"tz"`
	Slots []Slot `json:"slots"`
}

type ErrorResponse struct {
	Error   string       `json:"error"`
	Message string       `json:"message,omitempty"`
	Details []FieldError `json:"details,omitempty"`
}

type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func isInvalidError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return contains(msg, "invalid") || contains(msg, "validation") || contains(msg, "required")
}

func isNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	return contains(err.Error(), "not found")
}

func contains(s, sub string) bool {
	// simple contains without importing strings for small helper to avoid circular
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
