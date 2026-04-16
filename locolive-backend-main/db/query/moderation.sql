-- name: ListAllComments :many
SELECT 
    'post' as source,
    pc.id,
    pc.post_id as target_id,
    pc.user_id,
    u.username,
    pc.content,
    pc.is_flagged,
    pc.created_at
FROM post_comments pc
JOIN users u ON pc.user_id = u.id
UNION ALL
SELECT 
    'reel' as source,
    rc.id,
    rc.reel_id as target_id,
    rc.user_id,
    u.username,
    rc.content,
    rc.is_flagged,
    rc.created_at
FROM reel_comments rc
JOIN users u ON rc.user_id = u.id
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: UpdatePostCommentFlag :exec
UPDATE post_comments SET is_flagged = $2 WHERE id = $1;

-- name: UpdateReelCommentFlag :exec
UPDATE reel_comments SET is_flagged = $2 WHERE id = $1;

-- name: UpdateUserTrustScore :exec
UPDATE users SET trust_score = trust_score + $2 WHERE id = $1;

-- name: GetUserTrustScore :one
SELECT trust_score FROM users WHERE id = $1;
