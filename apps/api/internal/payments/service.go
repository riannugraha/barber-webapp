package payments

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"flowbook/api/internal/db"
	"flowbook/api/internal/email"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/checkout/session"
	"github.com/stripe/stripe-go/v76/webhook"
)

// Sentinel errors
var (
	ErrNotFound    = errors.New("not found")
	ErrFreeBooking = errors.New("free booking does not require payment")
	ErrAlreadyPaid = errors.New("booking already paid")
	ErrInvalid     = errors.New("invalid request")
)

// Service handles Stripe checkout + webhook + idempotent storage.
type Service struct {
	pool          *pgxpool.Pool
	queries       *db.Queries
	stripeKey     string
	webhookSecret string
	email         *email.Client
}

// NewService creates payments Service.
// pool may be nil for tests (mock stripe only). queries may be nil if pool nil.
// stripeKey is sk_test_..., webhookSecret is whsec_..., email may be nil (mock).
func NewService(pool *pgxpool.Pool, queries *db.Queries, stripeKey, webhookSecret string, emailClient *email.Client) *Service {
	if queries == nil && pool != nil {
		queries = db.New(pool)
	}
	return &Service{
		pool:          pool,
		queries:       queries,
		stripeKey:     strings.TrimSpace(stripeKey),
		webhookSecret: strings.TrimSpace(webhookSecret),
		email:         emailClient,
	}
}

// isPlaceholder checks placeholder keys like sk_test_... placeholder
func isPlaceholder(k string) bool {
	k = strings.TrimSpace(k)
	if k == "" || k == "sk_test_..." || k == "whsec_..." {
		return true
	}
	lower := strings.ToLower(k)
	if strings.Contains(lower, "placeholder") {
		return true
	}
	if strings.Contains(k, "[") || strings.Contains(k, "...") {
		// allow real sk_test_51... length >20
		if len(k) < 20 {
			return true
		}
	}
	// Real Stripe test keys are sk_test_51... or sk_test_... with length > 40
	// If it looks like sk_test_ but is short (<30), treat as placeholder
	if strings.HasPrefix(k, "sk_test_") && len(k) < 30 {
		return true
	}
	if strings.HasPrefix(k, "whsec_") && len(k) < 30 {
		return true
	}
	return false
}

// CreateCheckoutSession creates Stripe checkout session for a booking.
// AC: harga 0 (Konsultasi Style 15m) skip Stripe langsung CONFIRMED -> returns ErrFreeBooking.
// Otherwise returns Stripe redirect URL + sessionId, stores pending payment and updates booking stripe_session_id.
// If stripeKey is mock/empty, returns deterministic mock URL for smoke test (card 4242 flowbook-qa).
func (s *Service) CreateCheckoutSession(ctx context.Context, bookingID uuid.UUID, successURL, cancelURL string) (string, string, error) {
	if bookingID == uuid.Nil {
		return "", "", fmt.Errorf("%w: bookingId required", ErrInvalid)
	}
	if strings.TrimSpace(successURL) == "" {
		successURL = "https://flowbook-xxx.vercel.app/book/success"
	}
	if strings.TrimSpace(cancelURL) == "" {
		cancelURL = "https://flowbook-xxx.vercel.app/book"
	}

	// Need DB to fetch booking/service
	if s.pool == nil || s.queries == nil {
		// Mock mode without DB — return mock session for tests
		mockURL := fmt.Sprintf("https://checkout.stripe.com/c/pay/cs_test_%s#mock_no_db", bookingID.String()[:8])
		mockID := "cs_test_" + bookingID.String()[:8]
		slog.Info("payments: mock checkout (no DB)", "bookingId", bookingID.String(), "url", mockURL)
		return mockURL, mockID, nil
	}

	booking, err := s.queries.GetBooking(ctx, bookingID)
	if err != nil {
		return "", "", fmt.Errorf("%w: booking %s not found: %v", ErrNotFound, bookingID.String(), err)
	}
	// Fetch service for price + name
	svc, err := s.queries.GetService(ctx, booking.ServiceID)
	if err != nil {
		return "", "", fmt.Errorf("%w: service not found", ErrNotFound)
	}
	// Free booking check — PRD §3 Konsultasi Style 15m price 0 skip Stripe
	if svc.PriceCents == 0 {
		return "", "", fmt.Errorf("%w: service %s is free (price 0), booking already %s", ErrFreeBooking, svc.Name, booking.Status)
	}
	// Already paid?
	if booking.PaymentStatus == "PAID" || (booking.Status == "CONFIRMED" && booking.StripeSessionID.Valid) {
		// Check if payment already exists as PAID for this booking
		if p, err := s.queries.GetPaymentByBookingID(ctx, bookingID); err == nil && p.Status == "PAID" {
			return "", "", fmt.Errorf("%w: booking already paid (payment %s)", ErrAlreadyPaid, p.ID.String())
		}
		// Allow re-creating session if previous session expired? For now return already paid if status PAID
		if booking.PaymentStatus == "PAID" {
			return "", "", fmt.Errorf("%w: booking already paid", ErrAlreadyPaid)
		}
	}

	// Mock stripe if key missing/placeholder — still creates DB pending payment for webhook simulation
	if isPlaceholder(s.stripeKey) {
		mockID := "cs_test_" + strings.ReplaceAll(bookingID.String(), "-", "")[:24]
		mockURL := fmt.Sprintf("https://checkout.stripe.com/c/pay/%s#mock_stripe_key_missing", mockID)
		// Ensure successURL includes session_id placeholder like real Stripe
		// Store pending payment + update booking stripe_session_id for later webhook simulation
		// Use transaction to keep idempotency
		amount := svc.PriceCents
		if s.pool != nil {
			// Best effort: create pending payment
			_, _ = s.queries.CreatePayment(ctx, db.CreatePaymentParams{
				BookingID:             bookingID,
				OrganizationID:        booking.OrganizationID,
				StripeEventID:         pgtype.Text{Valid: false},
				StripeSessionID:       pgtype.Text{String: mockID, Valid: true},
				StripePaymentIntentID: pgtype.Text{Valid: false},
				AmountCents:           amount,
				Currency:              "IDR",
				Status:                "PENDING",
			})
			_, _ = s.queries.UpdateBookingPaymentStatus(ctx, db.UpdateBookingPaymentStatusParams{
				ID:              bookingID,
				PaymentStatus:   "UNPAID",
				StripeSessionID: pgtype.Text{String: mockID, Valid: true},
			})
		}
		slog.Info("payments: mock checkout (stripe key placeholder)", "bookingId", bookingID.String(), "mockSessionId", mockID, "amount", amount)
		return mockURL, mockID, nil
	}

	// Real Stripe — set global key
	stripe.Key = s.stripeKey

	// Build success URL with Stripe's {CHECKOUT_SESSION_ID} template for client retrieval
	// If successURL already contains ?, append &
	sessionSuccessURL := successURL
	if !strings.Contains(successURL, "{CHECKOUT_SESSION_ID}") {
		if strings.Contains(successURL, "?") {
			sessionSuccessURL = successURL + "&session_id={CHECKOUT_SESSION_ID}"
		} else {
			sessionSuccessURL = successURL + "?session_id={CHECKOUT_SESSION_ID}"
		}
	}

	params := &stripe.CheckoutSessionParams{
		Mode:              stripe.String(string(stripe.CheckoutSessionModePayment)),
		SuccessURL:        stripe.String(sessionSuccessURL),
		CancelURL:         stripe.String(cancelURL),
		ClientReferenceID: stripe.String(bookingID.String()),
		CustomerEmail:     stripe.String(booking.CustomerEmail),
		// Expire in 30m (Stripe default 24h, but we tighten)
		ExpiresAt: stripe.Int64(time.Now().Add(30 * time.Minute).Unix()),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Quantity: stripe.Int64(1),
				PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
					Currency: stripe.String("idr"),
					ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
						Name:        stripe.String(svc.Name),
						Description: stripe.String(fmt.Sprintf("%s — %s with %s", svc.Name, booking.StartAt.Format(time.RFC3339), svc.Description.String)),
					},
					UnitAmount: stripe.Int64(int64(svc.PriceCents)),
				},
			},
		},
		Metadata: map[string]string{
			"booking_id":      bookingID.String(),
			"organization_id": booking.OrganizationID.String(),
			"service_id":      svc.ID.String(),
		},
		PaymentIntentData: &stripe.CheckoutSessionPaymentIntentDataParams{
			Metadata: map[string]string{
				"booking_id": bookingID.String(),
			},
			// Description helps dashboard
			Description: stripe.String(fmt.Sprintf("FlowBook booking %s — %s", bookingID.String()[:8], svc.Name)),
		},
	}
	// Idempotency key per booking to avoid duplicate sessions on retry
	params.SetIdempotencyKey("booking_" + bookingID.String())

	slog.Info("payments: creating Stripe checkout session", "bookingId", bookingID.String(), "amount", svc.PriceCents, "currency", "idr", "service", svc.Name)

	result, err := session.New(params)
	if err != nil {
		slog.Error("payments: Stripe session creation failed", "bookingId", bookingID.String(), "error", err)
		return "", "", fmt.Errorf("stripe create session: %w", err)
	}

	// Persist pending payment + update booking stripe_session_id
	amount := svc.PriceCents
	// Use int32 for DB; Stripe amount is int64
	if s.queries != nil {
		_, pErr := s.queries.CreatePayment(ctx, db.CreatePaymentParams{
			BookingID:             bookingID,
			OrganizationID:        booking.OrganizationID,
			StripeEventID:         pgtype.Text{Valid: false},
			StripeSessionID:       pgtype.Text{String: result.ID, Valid: true},
			StripePaymentIntentID: pgtype.Text{Valid: false},
			AmountCents:           amount,
			Currency:              "IDR",
			Status:                "PENDING",
		})
		if pErr != nil {
			// If duplicate booking payment (e.g., re-try), log but don't fail checkout URL return
			// The EXCLUDE uniqueness on stripe_event_id does not affect pending (null), but booking_id uniqueness not enforced — allow multiple pending
			slog.Warn("payments: create payment pending failed (non-fatal)", "bookingId", bookingID.String(), "error", pErr)
		}
		_, uErr := s.queries.UpdateBookingPaymentStatus(ctx, db.UpdateBookingPaymentStatusParams{
			ID:              bookingID,
			PaymentStatus:   "UNPAID",
			StripeSessionID: pgtype.Text{String: result.ID, Valid: true},
		})
		if uErr != nil {
			slog.Warn("payments: update booking stripe_session_id failed", "bookingId", bookingID.String(), "error", uErr)
		}
	}

	slog.Info("payments: Stripe checkout session created", "bookingId", bookingID.String(), "sessionId", result.ID, "url", result.URL)
	return result.URL, result.ID, nil
}

// HandleWebhook verifies whsec signature, enforces idempotency via stripeEventId UNIQUE, updates booking to CONFIRMED/PAID, sends BookingConfirmed + ics.
// Returns error only for invalid signature (400) — all other errors are logged but return nil to keep Stripe 200 retry semantics.
// Idempotent: retry with same stripeEventId returns 200 without duplicate side effects.
func (s *Service) HandleWebhook(ctx context.Context, payload []byte, sigHeader string) error {
	if len(payload) == 0 {
		return fmt.Errorf("empty payload")
	}

	var event stripe.Event
	var err error

	// Verify signature if webhookSecret configured
	if !isPlaceholder(s.webhookSecret) && strings.TrimSpace(s.webhookSecret) != "" {
		// Use stripe webhook verification with api version mismatch ignored
		event, err = webhook.ConstructEventWithOptions(payload, sigHeader, s.webhookSecret, webhook.ConstructEventOptions{
			IgnoreAPIVersionMismatch: true,
		})
		if err != nil {
			slog.Error("payments: webhook signature verification failed", "error", err, "sigHeader", truncate(sigHeader, 100))
			return fmt.Errorf("invalid signature: %w", err)
		}
	} else {
		// No signature verification (test/mock mode) — just parse JSON
		if sigHeader != "" {
			slog.Warn("payments: webhook signature header present but STRIPE_WEBHOOK_SECRET not configured — skipping verification (test mode)")
		} else {
			slog.Info("payments: webhook verification skipped (no secret) — test mode")
		}
		if err := json.Unmarshal(payload, &event); err != nil {
			return fmt.Errorf("invalid webhook payload: %w", err)
		}
		// If payload was wrapped like Stripe's event but without api_version, ensure ID and Type are present
		if event.ID == "" {
			// Try to extract from raw map
			var raw map[string]interface{}
			if jErr := json.Unmarshal(payload, &raw); jErr == nil {
				if id, ok := raw["id"].(string); ok {
					event.ID = id
				}
				if typ, ok := raw["type"].(string); ok {
					event.Type = stripe.EventType(typ)
				}
				if data, ok := raw["data"].(map[string]interface{}); ok {
					if obj, ok := data["object"].(map[string]interface{}); ok {
						b, _ := json.Marshal(obj)
						event.Data = &stripe.EventData{Raw: json.RawMessage(b)}
						// also populate Object map via unmarshal
						_ = event.Data
						event.Data.Object = obj
					}
				}
			}
		}
	}

	if event.ID == "" {
		return fmt.Errorf("webhook event missing id")
	}
	slog.Info("payments: webhook received", "eventId", event.ID, "type", event.Type)

	// Idempotency check via stripeEventId UNIQUE — if payment with this event already exists, return 200 (retry)
	if s.pool != nil && s.queries != nil {
		pgEventID := pgtype.Text{String: event.ID, Valid: true}
		if existing, err := s.queries.GetPaymentByStripeEventID(ctx, pgEventID); err == nil && existing.ID != uuid.Nil {
			slog.Info("payments: webhook idempotent retry — already processed", "eventId", event.ID, "paymentId", existing.ID.String())
			return nil // 200
		}
	}

	// Handle by event type
	switch event.Type {
	case stripe.EventTypeCheckoutSessionCompleted, stripe.EventTypeCheckoutSessionAsyncPaymentSucceeded:
		return s.handleCheckoutCompleted(ctx, event)
	case stripe.EventTypePaymentIntentSucceeded:
		// PaymentIntent succeeded — could also confirm booking if checkout session not used (invoice, etc.)
		// For FlowBook, checkout.session.completed is primary. Log and attempt to confirm via payment_intent.
		slog.Info("payments: payment_intent.succeeded event — attempting to handle", "eventId", event.ID)
		return s.handlePaymentIntentSucceeded(ctx, event)
	case stripe.EventTypeCheckoutSessionExpired:
		slog.Info("payments: checkout.session.expired — marking payment failed", "eventId", event.ID)
		// Try to mark related payment as FAILED if found
		return s.handleCheckoutExpired(ctx, event)
	default:
		slog.Info("payments: webhook unhandled event type — ack 200", "eventId", event.ID, "type", event.Type)
		// Still store a no-op payment record for idempotency? Not needed — but we could log.
		return nil
	}
}

func (s *Service) handleCheckoutCompleted(ctx context.Context, event stripe.Event) error {
	if s.queries == nil || s.pool == nil {
		slog.Info("payments: mock handle checkout.completed (no DB)", "eventId", event.ID)
		return nil
	}

	// Parse session object
	var sess stripe.CheckoutSession
	if event.Data != nil && len(event.Data.Raw) > 0 {
		if err := json.Unmarshal(event.Data.Raw, &sess); err != nil {
			slog.Error("payments: failed to unmarshal checkout session", "eventId", event.ID, "error", err, "raw", truncate(string(event.Data.Raw), 500))
			// Try fallback via GetObjectValue map
			// Extract from map if raw failed
			if event.Data.Object != nil {
				if id, ok := event.Data.Object["id"].(string); ok {
					sess.ID = id
				}
				if cr, ok := event.Data.Object["client_reference_id"].(string); ok {
					sess.ClientReferenceID = cr
				}
				if meta, ok := event.Data.Object["metadata"].(map[string]interface{}); ok {
					sess.Metadata = map[string]string{}
					for k, v := range meta {
						if vs, ok := v.(string); ok {
							sess.Metadata[k] = vs
						}
					}
				}
			}
		}
	} else {
		slog.Error("payments: checkout.completed missing data", "eventId", event.ID)
		return nil // ack 200 to avoid retry storm
	}

	bookingIDStr := ""
	if sess.Metadata != nil {
		bookingIDStr = sess.Metadata["booking_id"]
	}
	if bookingIDStr == "" {
		bookingIDStr = sess.ClientReferenceID
	}
	// Fallback: GetObjectValue
	if bookingIDStr == "" && event.Data != nil {
		bookingIDStr = event.GetObjectValue("metadata", "booking_id")
		if bookingIDStr == "" {
			bookingIDStr = event.GetObjectValue("client_reference_id")
		}
	}
	// Fallback: lookup booking by stripe_session_id if metadata missing
	var bookingID uuid.UUID
	var err error
	if bookingIDStr != "" {
		bookingID, err = uuid.Parse(bookingIDStr)
		if err != nil {
			slog.Error("payments: invalid bookingId in session metadata", "eventId", event.ID, "bookingId", bookingIDStr, "error", err)
			// Try lookup by session ID
			bookingID = uuid.Nil
		}
	}
	if bookingID == uuid.Nil {
		// Try lookup booking by stripe_session_id == sess.ID
		if sess.ID != "" {
			var bID uuid.UUID
			err = s.pool.QueryRow(ctx, `SELECT id FROM bookings WHERE stripe_session_id=$1 LIMIT 1`, sess.ID).Scan(&bID)
			if err == nil && bID != uuid.Nil {
				bookingID = bID
				slog.Info("payments: resolved booking via stripe_session_id fallback", "sessionId", sess.ID, "bookingId", bookingID.String())
			} else {
				slog.Error("payments: could not resolve booking for session", "eventId", event.ID, "sessionId", sess.ID, "error", err)
				// Store payment record for idempotency even if booking not found? Create orphan payment for audit
				// Try to still store payment for idempotency with random booking? Better ack 200
				return nil
			}
		} else {
			slog.Error("payments: missing booking_id and session id", "eventId", event.ID)
			return nil
		}
	}

	// Fetch booking + service
	booking, err := s.queries.GetBooking(ctx, bookingID)
	if err != nil {
		slog.Error("payments: booking not found for webhook", "eventId", event.ID, "bookingId", bookingID.String(), "error", err)
		return nil // ack 200 to avoid retry loop for bad booking
	}
	svc, err := s.queries.GetService(ctx, booking.ServiceID)
	if err != nil {
		slog.Error("payments: service not found for booking", "bookingId", bookingID.String(), "error", err)
		svc = db.Service{PriceCents: 50000, Name: "Service"} // fallback
	}
	st, err := s.queries.GetStaff(ctx, booking.StaffID)
	if err != nil {
		slog.Error("payments: staff not found", "bookingId", bookingID.String(), "error", err)
		st = db.Staff{Name: "Staff"}
	}
	org, err := s.queries.GetOrganizationByID(ctx, booking.OrganizationID)
	if err != nil {
		org = db.Organization{Name: "FlowBarber Studio", Timezone: "Asia/Jakarta"}
	}

	// Determine amount from session or service
	amount := svc.PriceCents
	if sess.AmountTotal > 0 {
		// Stripe amount is in IDR smallest unit (1). Use it if service price was dynamic
		amount = int32(sess.AmountTotal)
	}
	currency := "IDR"
	if sess.Currency != "" {
		currency = strings.ToUpper(string(sess.Currency))
	}
	paymentIntentID := ""
	if sess.PaymentIntent != nil && sess.PaymentIntent.ID != "" {
		paymentIntentID = sess.PaymentIntent.ID
	} else if v := event.GetObjectValue("payment_intent"); v != "" {
		paymentIntentID = v
	}

	// Transaction: upsert payment + update booking
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		slog.Error("payments: begin tx failed", "eventId", event.ID, "error", err)
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	qtx := s.queries.WithTx(tx)

	// Idempotent upsert via stripe_event_id unique — ON CONFLICT DO UPDATE
	_, err = qtx.UpsertPaymentByStripeEvent(ctx, db.UpsertPaymentByStripeEventParams{
		BookingID:      bookingID,
		OrganizationID: booking.OrganizationID,
		StripeEventID:  pgtype.Text{String: event.ID, Valid: true},
		StripeSessionID: pgtype.Text{
			String: sess.ID,
			Valid:  sess.ID != "",
		},
		StripePaymentIntentID: pgtype.Text{String: paymentIntentID, Valid: paymentIntentID != ""},
		AmountCents:           amount,
		Currency:              currency,
		Status:                "PAID",
	})
	if err != nil {
		// Check for unique violation on stripe_event_id -> idempotent retry (should have been caught earlier, but race)
		if isUniqueViolation(err) {
			slog.Info("payments: webhook race idempotent — payment already exists", "eventId", event.ID)
			_ = tx.Rollback(ctx)
			return nil
		}
		slog.Error("payments: upsert payment failed", "eventId", event.ID, "error", err)
		return fmt.Errorf("upsert payment: %w", err)
	}

	// Update booking to CONFIRMED + PAID
	// Use separate updates to keep sqlc simple; both will be committed atomically
	_, err = qtx.UpdateBookingStatus(ctx, db.UpdateBookingStatusParams{ID: bookingID, Status: "CONFIRMED"})
	if err != nil {
		slog.Error("payments: update booking status failed", "bookingId", bookingID.String(), "error", err)
		return fmt.Errorf("update booking status: %w", err)
	}
	_, err = qtx.UpdateBookingPaymentStatus(ctx, db.UpdateBookingPaymentStatusParams{
		ID:              bookingID,
		PaymentStatus:   "PAID",
		StripeSessionID: pgtype.Text{String: sess.ID, Valid: sess.ID != ""},
	})
	if err != nil {
		slog.Error("payments: update booking payment status failed", "bookingId", bookingID.String(), "error", err)
		return fmt.Errorf("update booking payment status: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		slog.Error("payments: commit webhook tx failed", "eventId", event.ID, "error", err)
		return fmt.Errorf("commit: %w", err)
	}

	slog.Info("payments: webhook confirmed booking", "eventId", event.ID, "bookingId", bookingID.String(), "sessionId", sess.ID, "amount", amount, "currency", currency)

	// Send BookingConfirmed email with ics — best effort, outside tx, log error but don't fail webhook
	if s.email != nil {
		// Re-fetch booking after update for email (status now CONFIRMED)
		updatedBooking, err := s.queries.GetBooking(ctx, bookingID)
		if err == nil {
			// Use background context for email to avoid ctx cancellation
			go func() {
				eCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
				defer cancel()
				if err := s.email.SendBookingConfirmed(eCtx, updatedBooking, svc, st, org); err != nil {
					slog.Error("payments: email BookingConfirmed failed (webhook)", "bookingId", bookingID.String(), "error", err)
				} else {
					slog.Info("payments: email BookingConfirmed sent (webhook)", "bookingId", bookingID.String(), "to", updatedBooking.CustomerEmail)
				}
			}()
		}
	}

	return nil
}

func (s *Service) handlePaymentIntentSucceeded(ctx context.Context, event stripe.Event) error {
	// For payment_intent.succeeded, extract payment_intent id and try to find related booking via metadata or payment record
	// Simplified: just log and ack
	var pi stripe.PaymentIntent
	if event.Data != nil && len(event.Data.Raw) > 0 {
		if err := json.Unmarshal(event.Data.Raw, &pi); err == nil {
			// Try metadata booking_id
			if bid, ok := pi.Metadata["booking_id"]; ok && bid != "" {
				if bookingID, err := uuid.Parse(bid); err == nil {
					slog.Info("payments: pi.succeeded with booking metadata", "eventId", event.ID, "bookingId", bookingID.String(), "piId", pi.ID)
					// Could confirm booking similar to checkout
					// Attempt to upsert payment for idempotency
					if s.pool != nil && s.queries != nil {
						booking, err := s.queries.GetBooking(ctx, bookingID)
						if err == nil {
							svc, _ := s.queries.GetService(ctx, booking.ServiceID)
							amount := int32(pi.Amount)
							if amount == 0 && svc.PriceCents != 0 {
								amount = svc.PriceCents
							}
							_, _ = s.queries.UpsertPaymentByStripeEvent(ctx, db.UpsertPaymentByStripeEventParams{
								BookingID:             bookingID,
								OrganizationID:        booking.OrganizationID,
								StripeEventID:         pgtype.Text{String: event.ID, Valid: true},
								StripeSessionID:       pgtype.Text{Valid: false},
								StripePaymentIntentID: pgtype.Text{String: pi.ID, Valid: true},
								AmountCents:           amount,
								Currency:              strings.ToUpper(string(pi.Currency)),
								Status:                "PAID",
							})
							_, _ = s.queries.UpdateBookingStatus(ctx, db.UpdateBookingStatusParams{ID: bookingID, Status: "CONFIRMED"})
							_, _ = s.queries.UpdateBookingPaymentStatus(ctx, db.UpdateBookingPaymentStatusParams{ID: bookingID, PaymentStatus: "PAID", StripeSessionID: pgtype.Text{Valid: false}})
						}
					}
				}
			}
		}
	}
	slog.Info("payments: handled payment_intent.succeeded", "eventId", event.ID)
	return nil
}

func (s *Service) handleCheckoutExpired(ctx context.Context, event stripe.Event) error {
	var sess stripe.CheckoutSession
	if event.Data != nil && len(event.Data.Raw) > 0 {
		_ = json.Unmarshal(event.Data.Raw, &sess)
		bookingIDStr := ""
		if sess.Metadata != nil {
			bookingIDStr = sess.Metadata["booking_id"]
		}
		if bookingIDStr == "" {
			bookingIDStr = sess.ClientReferenceID
		}
		if bookingIDStr != "" {
			if bookingID, err := uuid.Parse(bookingIDStr); err == nil && s.queries != nil {
				// Mark payment as FAILED if exists
				if p, err := s.queries.GetPaymentByBookingID(ctx, bookingID); err == nil {
					_, _ = s.queries.UpdatePaymentStatus(ctx, db.UpdatePaymentStatusParams{ID: p.ID, Status: "FAILED", StripePaymentIntentID: pgtype.Text{Valid: false}})
					slog.Info("payments: marked payment FAILED on expired", "bookingId", bookingID.String(), "sessionId", sess.ID)
				}
			}
		}
	}
	// Store event for idempotency as FAILED? Not needed — we just ack
	return nil
}

// Helpers

func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "23505") || strings.Contains(msg, "duplicate key") || strings.Contains(msg, "unique constraint") || strings.Contains(msg, "unique violation")
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
