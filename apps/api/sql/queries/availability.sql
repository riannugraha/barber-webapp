-- name: ListAvailabilityByStaff :many
SELECT * FROM availability
WHERE staff_id = $1
ORDER BY day_of_week ASC, start_time ASC;

-- name: ListAvailabilityByOrganization :many
SELECT a.* FROM availability a
JOIN staff s ON s.id = a.staff_id
WHERE s.organization_id = $1
ORDER BY s.name, a.day_of_week;

-- name: GetAvailability :one
SELECT * FROM availability
WHERE id = $1
LIMIT 1;

-- name: CreateAvailability :one
INSERT INTO availability (staff_id, day_of_week, start_time, end_time)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: UpdateAvailability :one
UPDATE availability SET
    day_of_week = $2,
    start_time = $3,
    end_time = $4
WHERE id = $1
RETURNING *;

-- name: DeleteAvailability :exec
DELETE FROM availability WHERE id = $1;

-- name: DeleteAvailabilityByStaff :exec
DELETE FROM availability WHERE staff_id = $1;

-- name: ListOverridesByStaff :many
SELECT * FROM availability_overrides
WHERE staff_id = $1
ORDER BY date ASC;

-- name: ListOverridesByStaffAndRange :many
SELECT * FROM availability_overrides
WHERE staff_id = $1
  AND date BETWEEN $2 AND $3
ORDER BY date ASC;

-- name: GetOverride :one
SELECT * FROM availability_overrides
WHERE id = $1
LIMIT 1;

-- name: GetOverrideByStaffAndDate :one
SELECT * FROM availability_overrides
WHERE staff_id = $1 AND date = $2
LIMIT 1;

-- name: CreateOverride :one
INSERT INTO availability_overrides (staff_id, date, is_closed, start_time, end_time, reason)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: UpdateOverride :one
UPDATE availability_overrides SET
    is_closed = $2,
    start_time = $3,
    end_time = $4,
    reason = $5
WHERE id = $1
RETURNING *;

-- name: DeleteOverride :exec
DELETE FROM availability_overrides WHERE id = $1;

-- name: DeleteOverrideByStaffAndDate :exec
DELETE FROM availability_overrides WHERE staff_id = $1 AND date = $2;
