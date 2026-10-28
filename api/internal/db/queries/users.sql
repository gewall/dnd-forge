-- name: CreateUser :one
INSERT INTO users (email, password_hash, google_sub, name, role)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: UserByID :one
SELECT * FROM users WHERE id = $1;

-- name: UserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: UserByGoogleSub :one
SELECT * FROM users WHERE google_sub = $1;

-- name: LinkGoogleSub :one
UPDATE users SET google_sub = $2, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: SetUserRole :one
UPDATE users SET role = $2, updated_at = now()
WHERE id = $1
RETURNING *;

-- name: SetUserPassword :exec
UPDATE users SET password_hash = $2, updated_at = now()
WHERE id = $1;
