-- name: CreateRelationship :one
INSERT INTO relationships (
    user_id,
    target_user_id,
    type,
    status
) VALUES (
    $1, $2, $3, $4
) ON CONFLICT (user_id, target_user_id, type) DO UPDATE 
SET status = EXCLUDED.status, created_at = NOW()
RETURNING *;

-- name: DeleteRelationship :exec
DELETE FROM relationships
WHERE user_id = $1 AND target_user_id = $2 AND type = $3;

-- name: GetRelationship :one
SELECT * FROM relationships
WHERE user_id = $1 AND target_user_id = $2 AND type = $3
LIMIT 1;

-- name: ListFollowers :many
SELECT u.id, u.username, u.full_name, u.avatar_url, r.created_at
FROM relationships r
JOIN users u ON u.id = r.user_id
WHERE r.target_user_id = $1 AND r.type = 'follow' AND r.status = 'active'
ORDER BY r.created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListFollowing :many
SELECT u.id, u.username, u.full_name, u.avatar_url, r.created_at
FROM relationships r
JOIN users u ON u.id = r.target_user_id
WHERE r.user_id = $1 AND r.type = 'follow' AND r.status = 'active'
ORDER BY r.created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountFollowers :one
SELECT COUNT(*) FROM relationships
WHERE target_user_id = $1 AND type = 'follow' AND status = 'active';

-- name: CountFollowing :one
SELECT COUNT(*) FROM relationships
WHERE user_id = $1 AND type = 'follow' AND status = 'active';

-- name: ListPendingFollowRequests :many
SELECT 
    r.user_id as requester_id, 
    r.target_user_id as target_id, 
    r.status, 
    r.created_at,
    u.username,
    u.full_name,
    u.avatar_url
FROM relationships r
JOIN users u ON r.user_id = u.id
WHERE r.target_user_id = $1 
  AND r.type = 'follow'
  AND r.status = 'pending'
ORDER BY r.created_at DESC
LIMIT $2 OFFSET $3;

-- name: UpdateRelationshipStatus :one
UPDATE relationships
SET status = $3
WHERE user_id = $1 AND target_user_id = $2 AND type = 'follow'
RETURNING *;

-- name: ListSentFollowRequests :many
SELECT 
    r.user_id as requester_id, 
    r.target_user_id as target_id, 
    r.status, 
    r.created_at,
    u.username,
    u.full_name,
    u.avatar_url
FROM relationships r
JOIN users u ON r.target_user_id = u.id
WHERE r.user_id = $1 
  AND r.type = 'follow'
  AND r.status = 'pending'
ORDER BY r.created_at DESC
LIMIT $2 OFFSET $3;
