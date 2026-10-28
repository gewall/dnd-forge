-- name: CreateRefreshToken :exec
INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
VALUES ($1, $2, $3);

-- name: RefreshTokenByHash :one
SELECT * FROM refresh_tokens WHERE token_hash = $1
FOR UPDATE;

-- name: RevokeRefreshToken :exec
UPDATE refresh_tokens SET revoked_at = now()
WHERE id = $1 AND revoked_at IS NULL;

-- name: RevokeUserRefreshTokens :exec
UPDATE refresh_tokens SET revoked_at = now()
WHERE user_id = $1 AND revoked_at IS NULL;

-- name: DeleteExpiredRefreshTokens :exec
DELETE FROM refresh_tokens WHERE expires_at < now() - interval '7 days';
