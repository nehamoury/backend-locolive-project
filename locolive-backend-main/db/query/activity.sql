-- name: CreateActivityLog :one
INSERT INTO activity_logs (
    user_id,
    action_type,
    target_id,
    target_type,
    details
) VALUES (
    $1, $2, $3, $4, $5
) RETURNING *;

-- name: ListActivityLogs :many
SELECT al.*, u.username, u.avatar_url
FROM activity_logs al
JOIN users u ON al.user_id = u.id
ORDER BY al.created_at DESC
LIMIT $1 OFFSET $2;

-- name: DeleteActivityLog :exec
DELETE FROM activity_logs
WHERE id = $1;
