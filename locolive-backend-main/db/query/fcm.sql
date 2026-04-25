-- name: RegisterFCMToken :one
INSERT INTO user_fcm_tokens (
    user_id,
    token,
    device_type
) VALUES (
    $1, $2, $3
) ON CONFLICT (user_id, token) DO UPDATE SET
    last_used_at = now()
RETURNING *;

-- name: GetUserFCMTokens :many
SELECT token FROM user_fcm_tokens
WHERE user_id = $1;

-- name: RemoveFCMToken :exec
DELETE FROM user_fcm_tokens
WHERE user_id = $1 AND token = $2;

-- name: RemoveAllUserFCMTokens :exec
DELETE FROM user_fcm_tokens
WHERE user_id = $1;
