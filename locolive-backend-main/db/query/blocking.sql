-- name: BlockUser :one
INSERT INTO blocked_users (blocker_id, blocked_id)
VALUES ($1, $2)
ON CONFLICT (blocker_id, blocked_id) DO UPDATE 
SET created_at = NOW()
RETURNING *;

-- name: UnblockUser :exec
DELETE FROM blocked_users
WHERE blocker_id = $1 AND blocked_id = $2;

-- name: GetBlockedUsers :many
SELECT u.id, u.username, u.full_name, u.avatar_url, b.created_at as blocked_at
FROM blocked_users b
JOIN users u ON b.blocked_id = u.id
WHERE b.blocker_id = $1
ORDER BY b.created_at DESC;

-- name: IsUserBlocked :one
SELECT EXISTS (
    SELECT 1 FROM blocked_users
    WHERE blocker_id = $1 AND blocked_id = $2
);

-- name: ListAllBlocksAdmin :many
SELECT b.*, u1.username as blocker_username, u2.username as blocked_username
FROM blocked_users b
JOIN users u1 ON b.blocker_id = u1.id
JOIN users u2 ON b.blocked_id = u2.id
ORDER BY b.created_at DESC
LIMIT sqlc.arg(lim) OFFSET sqlc.arg(off);

-- name: GetBlocksForUser :many
SELECT b.*, u1.username as blocker_username, u2.username as blocked_username
FROM blocked_users b
JOIN users u1 ON b.blocker_id = u1.id
JOIN users u2 ON b.blocked_id = u2.id
WHERE b.blocker_id = $1 OR b.blocked_id = $1
ORDER BY b.created_at DESC;
