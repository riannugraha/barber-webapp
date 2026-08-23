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
GROUP BY 1
ORDER BY 1 ASC;

-- name: DashboardRevenueByGranularity :many
SELECT
    date_trunc($4::text, start_at) AS period,
    SUM(s.price_cents)::bigint AS revenue_cents,
    COUNT(*)::bigint AS bookings_count
FROM bookings b
JOIN services s ON s.id = b.service_id
WHERE b.organization_id = $1
  AND b.status IN ('CONFIRMED','COMPLETED')
  AND b.start_at BETWEEN $2 AND $3
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
  AND b.start_at BETWEEN $2 AND $3;

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
GROUP BY s.id, s.name, s.color
ORDER BY bookings_count DESC
LIMIT $4;

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
GROUP BY st.id, st.name
ORDER BY bookings_count DESC;

-- name: DashboardBookingsByHour :many
SELECT
    EXTRACT(HOUR FROM start_at)::int AS hour,
    COUNT(*)::bigint AS bookings_count
FROM bookings b
WHERE b.organization_id = $1
  AND b.start_at BETWEEN $2 AND $3
  AND b.status IN ('CONFIRMED','COMPLETED')
GROUP BY 1
ORDER BY 1 ASC;

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
GROUP BY b.customer_email, b.customer_name
ORDER BY bookings_count DESC
LIMIT $2;

-- name: DashboardRecentBookings :many
SELECT b.*, s.name AS service_name, st.name AS staff_name
FROM bookings b
JOIN services s ON s.id = b.service_id
JOIN staff st ON st.id = b.staff_id
WHERE b.organization_id = $1
ORDER BY b.created_at DESC
LIMIT $2;

-- name: DashboardOccupancy :one
SELECT
    COUNT(*) FILTER (WHERE status IN ('CONFIRMED','COMPLETED'))::float / NULLIF(COUNT(*),0)::float * 100 AS occupancy_pct
FROM bookings
WHERE organization_id = $1
  AND start_at BETWEEN $2 AND $3;
