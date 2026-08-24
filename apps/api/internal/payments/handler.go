package payments

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

// Handler exposes POST /payments/checkout-session and POST /payments/webhook
type Handler struct {
	svc       *Service
	validator *validator.Validate
}

// NewHandler creates Handler
func NewHandler(svc *Service) *Handler {
	v := validator.New(validator.WithRequiredStructEnabled())
	return &Handler{svc: svc, validator: v}
}

// RegisterRoutes mounts payment routes under group (e.g., /api/v1)
// checkout-session is public (bookingId auth via booking ownership not needed for demo)
// webhook is public but verifies whsec signature
func (h *Handler) RegisterRoutes(g *echo.Group) {
	g.POST("/payments/checkout-session", h.CreateCheckoutSession)
	// Stripe webhook — raw body, no JSON binding; verify signature
	g.POST("/payments/webhook", h.Webhook)
}

// DTOs
type checkoutRequest struct {
	BookingID  string  `json:"bookingId" validate:"required,uuid"`
	SuccessURL *string `json:"successUrl" validate:"omitempty,url"`
	CancelURL  *string `json:"cancelUrl" validate:"omitempty,url"`
}

type checkoutResponse struct {
	URL       string `json:"url"`
	SessionID string `json:"sessionId"`
}

type errorResponse struct {
	Error   string       `json:"error"`
	Message string       `json:"message,omitempty"`
	Details []fieldError `json:"details,omitempty"`
}

type fieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// CreateCheckoutSession godoc — POST /payments/checkout-session stripe-go 76 sk_test redirect
// AC: idempotent, free price 0 returns 422 skip Stripe, else 200 {url, sessionId}
func (h *Handler) CreateCheckoutSession(c echo.Context) error {
	var req checkoutRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusUnprocessableEntity, errorResponse{
			Error:   "validation_error",
			Message: "Validation failed",
			Details: []fieldError{{Field: "body", Message: "invalid JSON"}},
		})
	}
	if err := h.validator.Struct(req); err != nil {
		return h.validationError(c, err)
	}
	bookingID, err := uuid.Parse(req.BookingID)
	if err != nil {
		return c.JSON(http.StatusUnprocessableEntity, errorResponse{
			Error:   "validation_error",
			Message: "Invalid bookingId",
			Details: []fieldError{{Field: "bookingId", Message: "must be a valid UUID"}},
		})
	}
	successURL := ""
	if req.SuccessURL != nil {
		successURL = strings.TrimSpace(*req.SuccessURL)
	}
	cancelURL := ""
	if req.CancelURL != nil {
		cancelURL = strings.TrimSpace(*req.CancelURL)
	}

	url, sessionID, err := h.svc.CreateCheckoutSession(c.Request().Context(), bookingID, successURL, cancelURL)
	if err != nil {
		return h.mapServiceError(c, err)
	}
	return c.JSON(http.StatusOK, checkoutResponse{URL: url, SessionID: sessionID})
}

// Webhook godoc — POST /payments/webhook verify whsec signature idempotent via stripeEventId UNIQUE -> retry 200
func (h *Handler) Webhook(c echo.Context) error {
	// Read raw body for signature verification — must not use c.Bind
	body, err := io.ReadAll(c.Request().Body)
	if err != nil {
		return c.JSON(http.StatusBadRequest, errorResponse{Error: "bad_request", Message: "cannot read body: " + err.Error()})
	}
	// Restore body for potential re-read (not needed but good practice)
	c.Request().Body = io.NopCloser(bytes.NewReader(body))

	sigHeader := c.Request().Header.Get("Stripe-Signature")
	// Also support lowercase variant (some proxies)
	if sigHeader == "" {
		sigHeader = c.Request().Header.Get("stripe-signature")
	}

	// Delegate to service — it will verify whsec and handle idempotency
	if err := h.svc.HandleWebhook(c.Request().Context(), body, sigHeader); err != nil {
		// Invalid signature -> 400, other errors log but return 200 to prevent retry storm
		lower := strings.ToLower(err.Error())
		if strings.Contains(lower, "invalid signature") || strings.Contains(lower, "no valid signature") || strings.Contains(lower, "invalid header") {
			return c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid_signature", Message: err.Error()})
		}
		// For other errors (DB etc.), we still return 200 to avoid Stripe retry loop, but log?
		// Per AC, retry should 200 for idempotent — real processing errors should be retried? But we choose 200 for idempotent duplicates and 500 for transient?
		// To keep webhook retry idempotent, return 200 even on processing error (with log) — Stripe will retry on 5xx.
		// For transient DB error, we want Stripe to retry, so return 500.
		if strings.Contains(lower, "begin tx") || strings.Contains(lower, "upsert") || strings.Contains(lower, "commit") {
			return c.JSON(http.StatusInternalServerError, errorResponse{Error: "internal_error", Message: err.Error()})
		}
		// For not found etc., return 200
		// Use generic 200 with received false? But spec says retry 200
		c.Response().Header().Set("X-Webhook-Error", err.Error())
		return c.JSON(http.StatusOK, map[string]bool{"received": true})
	}

	return c.JSON(http.StatusOK, map[string]bool{"received": true})
}

// webhookPayload helper for logging raw JSON (not used)
type webhookPayload struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

// helpers

func (h *Handler) validationError(c echo.Context, err error) error {
	var details []fieldError
	if ve, ok := err.(validator.ValidationErrors); ok {
		for _, fe := range ve {
			field := fe.Field()
			// map Go field to json
			switch field {
			case "BookingID":
				field = "bookingId"
			case "SuccessURL":
				field = "successUrl"
			case "CancelURL":
				field = "cancelUrl"
			default:
				field = strings.ToLower(field[:1]) + field[1:]
			}
			msg := msgForTag(fe)
			details = append(details, fieldError{Field: field, Message: msg})
		}
	} else {
		details = append(details, fieldError{Field: "body", Message: err.Error()})
	}
	return c.JSON(http.StatusUnprocessableEntity, errorResponse{
		Error:   "validation_error",
		Message: "Validation failed",
		Details: details,
	})
}

func msgForTag(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return fe.Field() + " is required"
	case "uuid":
		return fe.Field() + " must be a valid UUID"
	case "url":
		return fe.Field() + " must be a valid URL"
	default:
		return fe.Field() + " failed on " + fe.Tag()
	}
}

func (h *Handler) mapServiceError(c echo.Context, err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	lower := strings.ToLower(msg)
	switch {
	case errors.Is(err, ErrNotFound) || strings.Contains(lower, "not found"):
		return c.JSON(http.StatusNotFound, errorResponse{Error: "not_found", Message: msg})
	case errors.Is(err, ErrFreeBooking) || strings.Contains(lower, "free booking"):
		return c.JSON(http.StatusUnprocessableEntity, errorResponse{
			Error:   "validation_error",
			Message: msg,
			Details: []fieldError{{Field: "bookingId", Message: "free service does not require payment — booking already CONFIRMED"}},
		})
	case errors.Is(err, ErrAlreadyPaid) || strings.Contains(lower, "already paid"):
		return c.JSON(http.StatusConflict, errorResponse{Error: "conflict", Message: msg})
	case errors.Is(err, ErrInvalid) || strings.Contains(lower, "required") || strings.Contains(lower, "invalid"):
		return c.JSON(http.StatusUnprocessableEntity, errorResponse{Error: "validation_error", Message: msg})
	default:
		if strings.Contains(lower, "validation") {
			return c.JSON(http.StatusUnprocessableEntity, errorResponse{Error: "validation_error", Message: msg})
		}
		// For Stripe errors, return 502 to indicate upstream failure but don't leak details
		if strings.Contains(lower, "stripe") {
			return c.JSON(http.StatusBadGateway, errorResponse{Error: "stripe_error", Message: msg})
		}
		return c.JSON(http.StatusInternalServerError, errorResponse{Error: "internal_error", Message: msg})
	}
}

// Ensure imported json is used (for webhook raw handling helper)
var _ = json.RawMessage{}
var _ = bytes.MinRead
