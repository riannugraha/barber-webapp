-- name: ListStaff :many
SELECT * FROM staff
WHERE organization_id = $1
  AND is_active = true
ORDER BY name ASC;

-- name: ListStaffAll :many
SELECT * FROM staff
WHERE organization_id = $1
ORDER BY created_at DESC;

-- name: GetStaff :one
SELECT * FROM staff
WHERE id = $1
LIMIT 1;

-- name: GetStaffByUserID :one
SELECT * FROM staff
WHERE user_id = $1
LIMIT 1;

-- name: GetStaffByEmail :one
SELECT * FROM staff
WHERE organization_id = $1 AND email = $2
LIMIT 1;

-- name: CreateStaff :one
INSERT INTO staff (
    organization_id, user_id, name, email, avatar_url, is_active
) VALUES (
    $1, $2, $3, $4, $5, $6
) RETURNING *;

-- name: UpdateStaff :one
UPDATE staff SET
    name = $2,
    email = $3,
    avatar_url = $4,
    is_active = $5,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteStaff :exec
DELETE FROM staff WHERE id = $1;

-- name: ListStaffByService :many
SELECT s.* FROM staff s
JOIN staff_services ss ON ss.staff_id = s.id
WHERE ss.service_id = $1
  AND s.is_active = true
ORDER BY s.name ASC;

-- name: ListStaffServices :many
SELECT * FROM staff_services
WHERE staff_id = $1;

-- name: AddStaffService :exec
INSERT INTO staff_services (staff_id, service_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: RemoveStaffService :exec
DELETE FROM staff_services
WHERE staff_id = $1 AND service_id = $2;

-- name: SetStaffServices :exec
DELETE FROM staff_services WHERE staff_id = $1;

-- name: CountStaff :one
SELECT COUNT(*) FROM staff WHERE organization_id = $1;
