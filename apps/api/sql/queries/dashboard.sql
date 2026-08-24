-- Dashboard aggregates — 5 row (PLAN.md T06) — all queries use DATE_TRUNC + GROUP BY in DB with index on start_at (idx_bookings_start_at, idx_bookings_org_status_time), not JS.
-- Store UTC (timestamptz), render in organization.timezone (Asia/Jakarta default). Availability engine uses tstzrange for overlap; dashboard uses BETWEEN + DATE_TRUNC for aggregates.
-- Staff scoping: OWNER full, STAFF filtered via optional staff_id param — pass '00000000-0000-0000-0000-000000000000'::uuid for no filter (OWNER), or real staff_id for STAFF.
-- All date ranges use BETWEEN with B-tree index on start_at; grouping via DATE_TRUNC uses hash aggregation in DB, not JS.

-- name: DashboardRevenueByDay :many
SELECT
    date_trunc('day', start_at)::date AS day,
    SUM(s.price_cents)::bigint AS revenue_cents,
    COUNT(*)::bigint AS bookings_count
FROM bookings b
JOIN services s ON s.id = b.service_id
WHERE b.organization_id = $1
  AND b.status IN ('CONFIRMED','COMPLETED')
  AND b.start_at BETWEEN $2 AND $3
  AND ($4::uuid = '00000000-0000-0000-0000-000000000000'::uuid OR b.staff_id = $4)
GROUP BY 1
ORDER BY 1 ASC;

-- name: DashboardRevenueByGranularity :many
SELECT
    date_trunc($4::text, start_at)::timestamptz AS period,
    SUM(s.price_cents)::bigint AS revenue_cents,
    COUNT(*)::bigint AS bookings_count
FROM bookings b
JOIN services s ON s.id = b.service_id
WHERE b.organization_id = $1
  AND b.status IN ('CONFIRMED','COMPLETED')
  AND b.start_at BETWEEN $2 AND $3
  AND ($5::uuid = '00000000-0000-0000-0000-000000000000'::uuid OR b.staff_id = $5)
GROUP BY 1
ORDER BY 1 ASC;

-- name: DashboardRevenueByGranularityTZ :many
SELECT
    date_trunc($4::text, (b.start_at AT TIME ZONE $5::text))::timestamptz AS period,
    SUM(s.price_cents)::bigint AS revenue_cents,
    COUNT(*)::bigint AS bookings_count
FROM bookings b
JOIN services s ON s.id = b.service_id
WHERE b.organization_id = $1
  AND b.status IN ('CONFIRMED','COMPLETED')
  AND b.start_at BETWEEN $2 AND $3
  AND ($6::uuid = '00000000-0000-0000-0000-000000000000'::uuid OR b.staff_id = $6)
GROUP BY 1
ORDER BY 1 ASC;

-- name: DashboardKPI :one
SELECT
    COUNT(*)::bigint AS total_bookings,
    COUNT(*) FILTER (WHERE status IN ('CONFIRMED','COMPLETED'))::bigint AS confirmed_bookings,
    COUNT(*) FILTER (WHERE status = 'CANCELLED')::bigint AS cancelled_bookings,
    COALESCE(SUM(s.price_cents) FILTER (WHERE b.status IN ('CONFIRMED','COMPLETED')), 0)::bigint AS total_revenue_cents,
    COALESCE(AVG(s.price_cents) FILTER (WHERE b.status IN ('CONFIRMED','COMPLETED')), 0)::bigint AS avg_ticket_cents
FROM bookings b
JOIN services s ON s.id = b.service_id
WHERE b.organization_id = $1
  AND b.start_at BETWEEN $2 AND $3
  AND ($4::uuid = '00000000-0000-0000-0000-000000000000'::uuid OR b.staff_id = $4);

-- name: DashboardTopServices :many
SELECT
    s.id,
    s.name,
    s.color,
    COUNT(*)::bigint AS bookings_count,
    SUM(s.price_cents)::bigint AS revenue_cents
FROM bookings b
JOIN services s ON s.id = b.service_id
WHERE b.organization_id = $1
  AND b.start_at BETWEEN $2 AND $3
  AND b.status IN ('CONFIRMED','COMPLETED')
  AND ($4::uuid = '00000000-0000-0000-0000-000000000000'::uuid OR b.staff_id = $4)
GROUP BY s.id, s.name, s.color
ORDER BY bookings_count DESC
LIMIT $5;

-- name: DashboardBookingsByStaff :many
SELECT
    st.id,
    st.name,
    COUNT(*)::bigint AS bookings_count,
    SUM(s.price_cents)::bigint AS revenue_cents
FROM bookings b
JOIN staff st ON st.id = b.staff_id
JOIN services s ON s.id = b.service_id
WHERE b.organization_id = $1
  AND b.start_at BETWEEN $2 AND $3
  AND b.status IN ('CONFIRMED','COMPLETED')
  AND ($4::uuid = '00000000-0000-0000-0000-000000000000'::uuid OR b.staff_id = $4)
GROUP BY st.id, st.name
ORDER BY bookings_count DESC;

-- name: DashboardBookingsByHour :many
SELECT
    EXTRACT(HOUR FROM (b.start_at AT TIME ZONE $4::text))::int AS hour,
    COUNT(*)::bigint AS bookings_count
FROM bookings b
WHERE b.organization_id = $1
  AND b.start_at BETWEEN $2 AND $3
  AND b.status IN ('CONFIRMED','COMPLETED')
  AND ($5::uuid = '00000000-0000-0000-0000-000000000000'::uuid OR b.staff_id = $5)
GROUP BY 1
ORDER BY 1 ASC;

-- name: DashboardHeatmap :many
SELECT
    EXTRACT(DOW FROM (b.start_at AT TIME ZONE $4::text))::int AS dow,
    EXTRACT(HOUR FROM (b.start_at AT TIME ZONE $4::text))::int AS hour,
    COUNT(*)::bigint AS bookings_count
FROM bookings b
WHERE b.organization_id = $1
  AND b.start_at BETWEEN $2 AND $3
  AND b.status IN ('CONFIRMED','COMPLETED')
  AND ($5::uuid = '00000000-0000-0000-0000-000000000000'::uuid OR b.staff_id = $5)
GROUP BY 1, 2
ORDER BY 1, 2 ASC;

-- name: DashboardTopCustomers :many
SELECT
    b.customer_email,
    b.customer_name,
    COUNT(*)::bigint AS bookings_count,
    SUM(s.price_cents)::bigint AS total_spent_cents,
    MAX(b.start_at) AS last_booking_at
FROM bookings b
JOIN services s ON s.id = b.service_id
WHERE b.organization_id = $1
  AND b.status IN ('CONFIRMED','COMPLETED')
  AND b.start_at BETWEEN $2 AND $3
  AND ($4::uuid = '00000000-0000-0000-0000-000000000000'::uuid OR b.staff_id = $4)
GROUP BY b.customer_email, b.customer_name
ORDER BY bookings_count DESC
LIMIT $5;

-- name: DashboardRecentBookings :many
SELECT b.id, b.organization_id, b.service_id, b.staff_id, b.customer_id, b.customer_name, b.customer_email, b.customer_phone, b.notes, b.start_at, b.end_at, b.status, b.payment_status, b.stripe_session_id, b.created_at, b.updated_at, s.name AS service_name, st.name AS staff_name
FROM bookings b
JOIN services s ON s.id = b.service_id
JOIN staff st ON st.id = b.staff_id
WHERE b.organization_id = $1
  AND ($2::uuid = '00000000-0000-0000-0000-000000000000'::uuid OR b.staff_id = $2)
ORDER BY b.created_at DESC
LIMIT $3;

-- name: DashboardOccupancy :one
SELECT
    COUNT(*) FILTER (WHERE status IN ('CONFIRMED','COMPLETED'))::float / NULLIF(COUNT(*),0)::float * 100 AS occupancy_pct
FROM bookings
WHERE organization_id = $1
  AND start_at BETWEEN $2 AND $3
  AND ($4::uuid = '00000000-0000-0000-0000-000000000000'::uuid OR staff_id = $4);

-- name: DashboardBusiestMonth :one
SELECT
    date_trunc('month', (b.start_at AT TIME ZONE $4::text))::date AS month,
    COUNT(*)::bigint AS bookings_count,
    SUM(s.price_cents)::bigint AS revenue_cents
FROM bookings b
JOIN services s ON s.id = b.service_id
WHERE b.organization_id = $1
  AND b.start_at BETWEEN $2 AND $3
  AND b.status IN ('CONFIRMED','COMPLETED')
  AND ($5::uuid = '00000000-0000-0000-0000-000000000000'::uuid OR b.staff_id = $5)
GROUP BY 1
ORDER BY bookings_count DESC
LIMIT 1;

-- name: DashboardCancelRate :one
SELECT
    COUNT(*) FILTER (WHERE status = 'CANCELLED')::float / NULLIF(COUNT(*),0)::float * 100 AS cancel_rate_pct
FROM bookings
WHERE organization_id = $1
  AND start_at BETWEEN $2 AND $3
  AND ($4::uuid = '00000000-0000-0000-0000-000000000000'::uuid OR staff_id = $4);
