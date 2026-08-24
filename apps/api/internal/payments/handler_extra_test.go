package payments

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestPayments_MapServiceError_Coverage(t *testing.T) {
	svc := NewService(nil, nil, "", "", nil)
	h := NewHandler(svc)
	e := echo.New()

	tests := []struct {
		err  error
		code int
	}{
		{ErrNotFound, http.StatusNotFound},
		{errors.New("booking not found"), http.StatusNotFound},
		{ErrFreeBooking, http.StatusUnprocessableEntity},
		{errors.New("free booking does not require payment"), http.StatusUnprocessableEntity},
		{ErrAlreadyPaid, http.StatusConflict},
		{errors.New("already paid"), http.StatusConflict},
		{ErrInvalid, http.StatusUnprocessableEntity},
		{errors.New("invalid request"), http.StatusUnprocessableEntity},
		{errors.New("stripe create session failed"), http.StatusBadGateway},
		{errors.New("some internal error"), http.StatusInternalServerError},
	}
	for _, tc := range tests {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)
		_ = h.mapServiceError(c, tc.err)
		assert.Equal(t, tc.code, rec.Code, "err %v", tc.err)
	}
	// nil error
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	err := h.mapServiceError(c, nil)
	assert.Nil(t, err)
}

func TestPayments_ValidationError_MsgForTag(t *testing.T) {
	svc := NewService(nil, nil, "", "", nil)
	h := NewHandler(svc)
	e := echo.New()
	// Test validationError via handler's CreateCheckoutSession with invalid payload
	// This will internally call validationError and msgForTag
	req := httptest.NewRequest(http.MethodPost, "/payments/checkout-session", bytes.NewReader([]byte(`{"bookingId":"bad"}`)))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	_ = h.CreateCheckoutSession(c)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)

	_ = e
}

func TestPayments_Handler_CreateCheckoutSession_SuccessViaDB(t *testing.T) {
	// Use real DB to test handler success path
	_, _, _, bookingID := setupPaymentData(t)
	svc := NewService(testPool, testQueries, "", "", nil)
	h := NewHandler(svc)
	e := echo.New()
	body := `{"bookingId":"` + bookingID.String() + `"}`
	req := httptest.NewRequest(http.MethodPost, "/payments/checkout-session", bytes.NewReader([]byte(body)))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	_ = h.CreateCheckoutSession(c)
	// Should be 200 with mock URL (since no stripe key)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "checkout.stripe.com")
}
