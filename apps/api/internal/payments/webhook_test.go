package payments

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"flowbook/api/internal/db"
	"flowbook/api/internal/testhelpers"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	testPool    *pgxpool.Pool
	testQueries *db.Queries
	testCtr     *testhelpers.PostgresContainer
)

func TestMain(m *testing.M) {
	ctx := context.Background()
	ctr, err := testhelpers.CreatePostgresContainer(ctx)
	if err != nil {
		panic("payments TestMain container failed: " + err.Error())
	}
	testCtr = ctr
	testPool = ctr.Pool
	testQueries = db.New(testPool)
	code := m.Run()
	_ = ctr.Terminate(ctx)
	os.Exit(code)
}

func setupPaymentData(t *testing.T) (orgID, serviceID, staffID, bookingID uuid.UUID) {
	t.Helper()
	ctx := context.Background()
	_, err := testPool.Exec(ctx, `TRUNCATE bookings, payments, customers RESTART IDENTITY CASCADE`)
	require.NoError(t, err)
	_, err = testPool.Exec(ctx, `TRUNCATE staff_services, availability, staff, services, organizations RESTART IDENTITY CASCADE`)
	require.NoError(t, err)

	orgID = uuid.New()
	_, err = testPool.Exec(ctx, `INSERT INTO organizations (id, name, slug, timezone) VALUES ($1,$2,$3,$4)`, orgID, "Pay Org", "pay-org-"+orgID.String()[:6], "Asia/Jakarta")
	require.NoError(t, err)

	serviceID = uuid.New()
	_, err = testPool.Exec(ctx, `INSERT INTO services (id, organization_id, name, duration_minutes, buffer_minutes, price_cents) VALUES ($1,$2,$3,$4,$5,$6)`, serviceID, orgID, "Classic Cut", 30, 10, 85000)
	require.NoError(t, err)

	staffID = uuid.New()
	_, err = testPool.Exec(ctx, `INSERT INTO staff (id, organization_id, name, email) VALUES ($1,$2,$3,$4)`, staffID, orgID, "Andi", "andi@pay.test")
	require.NoError(t, err)

	_, err = testPool.Exec(ctx, `INSERT INTO staff_services (staff_id, service_id) VALUES ($1,$2)`, staffID, serviceID)
	require.NoError(t, err)

	bookingID = uuid.New()
	start := time.Now().Add(2 * time.Hour).UTC()
	end := start.Add(40 * time.Minute)
	_, err = testPool.Exec(ctx, `INSERT INTO bookings (id, organization_id, service_id, staff_id, customer_name, customer_email, start_at, end_at, status, payment_status) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`, bookingID, orgID, serviceID, staffID, "Pay Customer", "pay@test.com", start, end, "PENDING", "UNPAID")
	require.NoError(t, err)

	return orgID, serviceID, staffID, bookingID
}

func TestHandleWebhook_IdempotentRetry200(t *testing.T) {
	require.NotNil(t, testPool)
	_, _, _, bookingID := setupPaymentData(t)
	ctx := context.Background()

	svc := NewService(testPool, testQueries, "", "", nil) // empty keys => mock mode, no signature verify
	// Build stripe event payload checkout.session.completed with metadata booking_id
	eventID := "evt_test_idempotent_" + uuid.New().String()[:8]
	payloadMap := map[string]interface{}{
		"id":   eventID,
		"type": "checkout.session.completed",
		"data": map[string]interface{}{
			"object": map[string]interface{}{
				"id": "cs_test_123",
				"object": "checkout.session",
				"client_reference_id": bookingID.String(),
				"metadata": map[string]interface{}{
					"booking_id": bookingID.String(),
				},
				"amount_total": 85000,
				"currency": "idr",
				"payment_intent": "pi_test_123",
			},
		},
	}
	payload, _ := json.Marshal(payloadMap)

	// First call should succeed and confirm booking
	err := svc.HandleWebhook(ctx, payload, "")
	require.NoError(t, err)
	// Verify booking now CONFIRMED + PAID
	var status, payStatus string
	err = testPool.QueryRow(ctx, `SELECT status, payment_status FROM bookings WHERE id=$1`, bookingID).Scan(&status, &payStatus)
	require.NoError(t, err)
	assert.Equal(t, "CONFIRMED", status)
	assert.Equal(t, "PAID", payStatus)
	// Verify payment inserted with stripe_event_id
	var payCount int64
	err = testPool.QueryRow(ctx, `SELECT COUNT(*) FROM payments WHERE stripe_event_id=$1`, eventID).Scan(&payCount)
	require.NoError(t, err)
	assert.Equal(t, int64(1), payCount)

	// Second call with same eventID -> idempotent, should return nil (200) and not duplicate
	err = svc.HandleWebhook(ctx, payload, "")
	require.NoError(t, err)
	err = testPool.QueryRow(ctx, `SELECT COUNT(*) FROM payments WHERE stripe_event_id=$1`, eventID).Scan(&payCount)
	require.NoError(t, err)
	assert.Equal(t, int64(1), payCount, "idempotent retry should not create duplicate payment")
	// Booking still confirmed
	err = testPool.QueryRow(ctx, `SELECT status FROM bookings WHERE id=$1`, bookingID).Scan(&status)
	require.NoError(t, err)
	assert.Equal(t, "CONFIRMED", status)
}

func TestHandleWebhook_EmptyPayload(t *testing.T) {
	svc := NewService(nil, nil, "", "", nil)
	err := svc.HandleWebhook(context.Background(), []byte{}, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty payload")
}

func TestHandleWebhook_MissingBooking(t *testing.T) {
	// Event with unknown booking_id -> should ack 200 (not error) but not create payment? Our service returns nil after not finding booking
	svc := NewService(testPool, testQueries, "", "", nil)
	eventID := "evt_missing_" + uuid.New().String()[:8]
	fakeBooking := uuid.New().String()
	payloadMap := map[string]interface{}{
		"id": eventID,
		"type": "checkout.session.completed",
		"data": map[string]interface{}{
			"object": map[string]interface{}{
				"id": "cs_test_missing",
				"client_reference_id": fakeBooking,
				"metadata": map[string]interface{}{"booking_id": fakeBooking},
			},
		},
	}
	payload, _ := json.Marshal(payloadMap)
	err := svc.HandleWebhook(context.Background(), payload, "")
	// Should not return signature error, should ack 200 (nil) even though booking not found
	require.NoError(t, err)
}

func TestCreateCheckoutSession_FreeBooking(t *testing.T) {
	require.NotNil(t, testPool)
	ctx := context.Background()
	// Create free service
	orgID := uuid.New()
	_, err := testPool.Exec(ctx, `INSERT INTO organizations (id, name, slug, timezone) VALUES ($1,$2,$3,$4)`, orgID, "Free Org", "free-org-"+orgID.String()[:6], "Asia/Jakarta")
	require.NoError(t, err)
	svcID := uuid.New()
	_, err = testPool.Exec(ctx, `INSERT INTO services (id, organization_id, name, duration_minutes, buffer_minutes, price_cents) VALUES ($1,$2,$3,$4,$5,$6)`, svcID, orgID, "Konsultasi Style 15m", 15, 5, 0)
	require.NoError(t, err)
	staffID := uuid.New()
	_, err = testPool.Exec(ctx, `INSERT INTO staff (id, organization_id, name) VALUES ($1,$2,$3)`, staffID, orgID, "Bayu")
	require.NoError(t, err)
	bookingID := uuid.New()
	start := time.Now().Add(24 * time.Hour).UTC()
	end := start.Add(20 * time.Minute)
	_, err = testPool.Exec(ctx, `INSERT INTO bookings (id, organization_id, service_id, staff_id, customer_name, customer_email, start_at, end_at, status, payment_status) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`, bookingID, orgID, svcID, staffID, "Free Customer", "free@test.com", start, end, "CONFIRMED", "PAID")
	require.NoError(t, err)

	svc := NewService(testPool, testQueries, "sk_test_placeholder", "whsec_placeholder", nil)
	_, _, err = svc.CreateCheckoutSession(ctx, bookingID, "", "")
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrFreeBooking)
}

func TestCreateCheckoutSession_MockWithoutDB(t *testing.T) {
	svc := NewService(nil, nil, "", "", nil)
	bookingID := uuid.New()
	url, sid, err := svc.CreateCheckoutSession(context.Background(), bookingID, "https://example.com/success", "https://example.com/cancel")
	require.NoError(t, err)
	assert.Contains(t, url, "cs_test_")
	assert.Contains(t, sid, "cs_test_")
}

func TestWebhookHandler_Httptest_200And400(t *testing.T) {
	_, _, _, bookingID := setupPaymentData(t)
	svc := NewService(testPool, testQueries, "", "", nil)
	h := NewHandler(svc)
	e := echo.New()
	// Success case via HTTP
	eventID := "evt_http_" + uuid.New().String()[:8]
	payloadMap := map[string]interface{}{
		"id": eventID,
		"type": "checkout.session.completed",
		"data": map[string]interface{}{
			"object": map[string]interface{}{
				"id": "cs_http_123",
				"client_reference_id": bookingID.String(),
				"metadata": map[string]interface{}{"booking_id": bookingID.String()},
				"amount_total": 85000,
				"currency": "idr",
			},
		},
	}
	payload, _ := json.Marshal(payloadMap)
	req := httptest.NewRequest(http.MethodPost, "/payments/webhook", bytes.NewReader(payload))
	req.Header.Set("Stripe-Signature", "t=123,v1=abc")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	// Handler will call svc.HandleWebhook with body and sigHeader; since svc has no secret, it skips verify
	err := h.Webhook(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), "received")

	// Empty body -> 400
	req2 := httptest.NewRequest(http.MethodPost, "/payments/webhook", bytes.NewReader([]byte{}))
	rec2 := httptest.NewRecorder()
	c2 := e.NewContext(req2, rec2)
	_ = h.Webhook(c2)
	assert.Equal(t, http.StatusBadRequest, rec2.Code)

	// Invalid signature when secret configured should be 400
	svcWithSecret := NewService(testPool, testQueries, "sk_test_51abcdefghijklmnopqrstuvwxyz0123456789ABCDEF", "whsec_1J2F8e2eZvKYlo2C4a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6", nil)
	h2 := NewHandler(svcWithSecret)
	// Use fresh payload with new eventID so idempotency not triggered
	eventID2 := "evt_invalid_" + uuid.New().String()[:8]
	payloadMap2 := map[string]interface{}{
		"id": eventID2,
		"type": "checkout.session.completed",
		"data": map[string]interface{}{
			"object": map[string]interface{}{
				"id": "cs_invalid_123",
				"client_reference_id": bookingID.String(),
				"metadata": map[string]interface{}{"booking_id": bookingID.String()},
			},
		},
	}
	payload2, _ := json.Marshal(payloadMap2)
	req3 := httptest.NewRequest(http.MethodPost, "/payments/webhook", bytes.NewReader(payload2))
	req3.Header.Set("Stripe-Signature", "invalid")
	rec3 := httptest.NewRecorder()
	c3 := e.NewContext(req3, rec3)
	_ = h2.Webhook(c3)
	assert.Equal(t, http.StatusBadRequest, rec3.Code)
}

func TestWebhookHandler_CheckoutValidation(t *testing.T) {
	svc := NewService(nil, nil, "", "", nil)
	h := NewHandler(svc)
	e := echo.New()
	// Bad JSON for checkout session
	req := httptest.NewRequest(http.MethodPost, "/payments/checkout-session", bytes.NewReader([]byte(`{bad`)))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	_ = h.CreateCheckoutSession(c)
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)

	// Missing bookingId
	req2 := httptest.NewRequest(http.MethodPost, "/payments/checkout-session", bytes.NewReader([]byte(`{}`)))
	req2.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec2 := httptest.NewRecorder()
	c2 := e.NewContext(req2, rec2)
	_ = h.CreateCheckoutSession(c2)
	assert.Equal(t, http.StatusUnprocessableEntity, rec2.Code)
}

func TestPaymentsHandler_RegisterRoutes(t *testing.T) {
	svc := NewService(nil, nil, "", "", nil)
	h := NewHandler(svc)
	e := echo.New()
	g := e.Group("/api/v1")
	h.RegisterRoutes(g)
	found := false
	for _, r := range e.Routes() {
		if r.Path == "/api/v1/payments/webhook" {
			found = true
		}
	}
	assert.True(t, found)
}

func TestIsPlaceholder(t *testing.T) {
	assert.True(t, isPlaceholder(""))
	assert.True(t, isPlaceholder("sk_test_..."))
	assert.True(t, isPlaceholder("whsec_..."))
	assert.True(t, isPlaceholder("sk_test_123"))
	assert.False(t, isPlaceholder("sk_test_51J2F8e2eZvKYlo2C4a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0"))
	assert.False(t, isPlaceholder("whsec_1J2F8e2eZvKYlo2C4a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6"))
}

func TestHandleWebhook_UnhandledEventType(t *testing.T) {
	svc := NewService(testPool, testQueries, "", "", nil)
	payloadMap := map[string]interface{}{
		"id": "evt_unhandled_" + uuid.New().String()[:8],
		"type": "customer.created",
		"data": map[string]interface{}{
			"object": map[string]interface{}{"id": "cus_123"},
		},
	}
	payload, _ := json.Marshal(payloadMap)
	err := svc.HandleWebhook(context.Background(), payload, "")
	require.NoError(t, err)
}

func TestCheckoutSession_AlreadyPaid(t *testing.T) {
	require.NotNil(t, testPool)
	orgID, svcID, _, bookingID := setupPaymentData(t)
	ctx := context.Background()
	// First create checkout session (mock)
	svc := NewService(testPool, testQueries, "", "", nil)
	url, _, err := svc.CreateCheckoutSession(ctx, bookingID, "", "")
	require.NoError(t, err)
	assert.Contains(t, url, "checkout.stripe.com")
	// Mark booking as PAID + payment PAID
	_, err = testPool.Exec(ctx, `UPDATE bookings SET payment_status='PAID', status='CONFIRMED' WHERE id=$1`, bookingID)
	require.NoError(t, err)
	_, err = testPool.Exec(ctx, `INSERT INTO payments (id, booking_id, organization_id, amount_cents, currency, status) VALUES ($1,$2,$3,$4,$5,$6)`, uuid.New(), bookingID, orgID, 85000, "IDR", "PAID")
	require.NoError(t, err)
	// Need to fetch org for update? Actually CreateCheckoutSession checks GetPaymentByBookingID
	_, _, err = svc.CreateCheckoutSession(ctx, bookingID, "", "")
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrAlreadyPaid)
	_ = svcID
}

func TestWebhook_WithInvalidJSON(t *testing.T) {
	svc := NewService(testPool, testQueries, "", "", nil)
	err := svc.HandleWebhook(context.Background(), []byte(`{invalid`), "")
	assert.Error(t, err)
}

// Ensure imports used
var _ = pgtype.Text{}
var _ = json.RawMessage{}
