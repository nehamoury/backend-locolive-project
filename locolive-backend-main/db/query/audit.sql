-- name: CreateUserAuditLog :one
INSERT INTO user_activity_logs (
    user_id, action, details, ip_address, user_agent
) VALUES (
    $1, $2, $3, $4, $5
) RETURNING *;

-- name: GetUserAuditLogs :many
SELECT * FROM user_activity_logs
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetAuditLogsByAction :many
SELECT * FROM user_activity_logs
WHERE action = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;
