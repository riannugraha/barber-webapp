-- name: ListServices :many
SELECT * FROM services
WHERE organization_id = $1
  AND is_active = true
ORDER BY name ASC;

-- name: ListServicesAll :many
SELECT * FROM services
WHERE organization_id = $1
ORDER BY created_at DESC;

-- name: GetService :one
SELECT * FROM services
WHERE id = $1
LIMIT 1;

-- name: GetServiceByName :one
SELECT * FROM services
WHERE organization_id = $1 AND name = $2
LIMIT 1;

-- name: CreateService :one
INSERT INTO services (
    organization_id, name, description, duration_minutes, buffer_minutes, price_cents, color, is_active
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
) RETURNING *;

-- name: UpdateService :one
UPDATE services SET
    name = $2,
    description = $3,
    duration_minutes = $4,
    buffer_minutes = $5,
    price_cents = $6,
    color = $7,
    is_active = $8,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteService :exec
DELETE FROM services WHERE id = $1;

-- name: CountServices :one
SELECT COUNT(*) FROM services WHERE organization_id = $1;
