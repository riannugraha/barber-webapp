-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1 LIMIT 1;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1 LIMIT 1;

-- name: CreateUser :one
INSERT INTO users (organization_id, email, password_hash, name, role)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: UpdateUser :one
UPDATE users SET
    name = $2,
    role = $3,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: ListUsersByOrganization :many
SELECT * FROM users WHERE organization_id = $1 ORDER BY created_at DESC;

-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetRefreshTokenByHash :one
SELECT * FROM refresh_tokens WHERE token_hash = $1 LIMIT 1;

-- name: RevokeRefreshToken :exec
UPDATE refresh_tokens SET revoked = true WHERE token_hash = $1;

-- name: DeleteExpiredRefreshTokens :exec
DELETE FROM refresh_tokens WHERE expires_at < now() OR revoked = true;

-- name: ListRefreshTokensByUser :many
SELECT * FROM refresh_tokens WHERE user_id = $1 ORDER BY created_at DESC;

-- name: GetOrganizationByID :one
SELECT * FROM organizations WHERE id = $1 LIMIT 1;

-- name: GetOrganizationBySlug :one
SELECT * FROM organizations WHERE slug = $1 LIMIT 1;

-- name: CreateOrganization :one
INSERT INTO organizations (name, slug, timezone, logo_url)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: ListCustomers :many
SELECT * FROM customers
WHERE organization_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetCustomerByEmail :one
SELECT * FROM customers
WHERE organization_id = $1 AND email = $2
LIMIT 1;

-- name: CreateCustomer :one
INSERT INTO customers (organization_id, email, name, phone)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: UpsertCustomer :one
INSERT INTO customers (organization_id, email, name, phone)
VALUES ($1, $2, $3, $4)
ON CONFLICT (organization_id, email) DO UPDATE SET
    name = EXCLUDED.name,
    phone = COALESCE(EXCLUDED.phone, customers.phone),
    updated_at = now()
RETURNING *;
