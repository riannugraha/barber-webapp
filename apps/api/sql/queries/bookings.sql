-- name: ListBookings :many
SELECT * FROM bookings
WHERE organization_id = $1
  AND ($2::timestamptz IS NULL OR start_at >= $2)
  AND ($3::timestamptz IS NULL OR start_at <= $3)
  AND ($4::text IS NULL OR status = $4)
  AND ($5::uuid IS NULL OR staff_id = $5)
ORDER BY start_at DESC
LIMIT $6 OFFSET $7;

-- name: ListBookingsByStaff :many
SELECT * FROM bookings
WHERE staff_id = $1
  AND ($2::timestamptz IS NULL OR start_at >= $2)
  AND ($3::timestamptz IS NULL OR start_at <= $3)
ORDER BY start_at ASC;

-- name: ListBookingsByCustomerEmail :many
SELECT * FROM bookings
WHERE customer_email = $1
ORDER BY start_at DESC;

-- name: GetBooking :one
SELECT * FROM bookings
WHERE id = $1
LIMIT 1;

-- name: GetBookingByIDAndOrg :one
SELECT * FROM bookings
WHERE id = $1 AND organization_id = $2
LIMIT 1;

-- name: CreateBooking :one
INSERT INTO bookings (
    organization_id, service_id, staff_id, customer_id,
    customer_name, customer_email, customer_phone, notes,
    start_at, end_at, status, payment_status, stripe_session_id
) VALUES (
    $1, $2, $3, $4,
    $5, $6, $7, $8,
    $9, $10, $11, $12, $13
) RETURNING *;

-- name: UpdateBookingStatus :one
UPDATE bookings SET
    status = $2,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: UpdateBookingPaymentStatus :one
UPDATE bookings SET
    payment_status = $2,
    stripe_session_id = COALESCE($3, stripe_session_id),
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: RescheduleBooking :one
UPDATE bookings SET
    staff_id = $2,
    start_at = $3,
    end_at = $4,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: CancelBooking :one
UPDATE bookings SET
    status = 'CANCELLED',
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: ListOverlappingBookings :many
SELECT * FROM bookings
WHERE staff_id = $1
  AND status IN ('PENDING','CONFIRMED')
  AND tstzrange(start_at, end_at) && tstzrange($2::timestamptz, $3::timestamptz);

-- name: CountBookingsByStatus :many
SELECT status, COUNT(*)::bigint AS count
FROM bookings
WHERE organization_id = $1
GROUP BY status;

-- name: DeleteBooking :exec
DELETE FROM bookings WHERE id = $1;

-- name: CountBookings :one
SELECT COUNT(*) FROM bookings WHERE organization_id = $1;
