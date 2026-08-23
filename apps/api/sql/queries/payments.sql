-- name: GetPayment :one
SELECT * FROM payments
WHERE id = $1
LIMIT 1;

-- name: GetPaymentByBookingID :one
SELECT * FROM payments
WHERE booking_id = $1
LIMIT 1;

-- name: GetPaymentByStripeEventID :one
SELECT * FROM payments
WHERE stripe_event_id = $1
LIMIT 1;

-- name: GetPaymentByStripeSessionID :one
SELECT * FROM payments
WHERE stripe_session_id = $1
LIMIT 1;

-- name: ListPaymentsByOrganization :many
SELECT * FROM payments
WHERE organization_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: CreatePayment :one
INSERT INTO payments (
    booking_id, organization_id, stripe_event_id, stripe_session_id, stripe_payment_intent_id,
    amount_cents, currency, status
) VALUES (
    $1, $2, $3, $4, $5,
    $6, $7, $8
) RETURNING *;

-- name: UpsertPaymentByStripeEvent :one
INSERT INTO payments (
    booking_id, organization_id, stripe_event_id, stripe_session_id, stripe_payment_intent_id,
    amount_cents, currency, status
) VALUES (
    $1, $2, $3, $4, $5,
    $6, $7, $8
) ON CONFLICT (stripe_event_id) DO UPDATE SET
    status = EXCLUDED.status,
    stripe_payment_intent_id = COALESCE(EXCLUDED.stripe_payment_intent_id, payments.stripe_payment_intent_id),
    updated_at = now()
RETURNING *;

-- name: UpdatePaymentStatus :one
UPDATE payments SET
    status = $2,
    stripe_payment_intent_id = COALESCE($3, stripe_payment_intent_id),
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: UpdatePaymentByStripeEvent :one
UPDATE payments SET
    status = $2,
    updated_at = now()
WHERE stripe_event_id = $1
RETURNING *;

-- name: CountPayments :one
SELECT COUNT(*) FROM payments WHERE organization_id = $1;
