-- name: CreatePasswordReset :one
INSERT INTO password_resets (
    user_id,
    token,
    expires_at
) VALUES (
    $1, $2, $3
) RETURNING *;

-- name: GetPasswordResetByToken :one
SELECT * FROM password_resets
WHERE token = $1 
AND expires_at > now()
LIMIT 1;

-- name: DeletePasswordResetByToken :exec
DELETE FROM password_resets
WHERE token = $1;

-- name: DeleteExpiredPasswordResets :exec
DELETE FROM password_resets
WHERE expires_at < now();

-- name: DeleteUserPasswordResets :exec
DELETE FROM password_resets
WHERE user_id = $1;
