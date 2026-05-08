-- name: CreateMention :one
INSERT INTO mentions (
    entity_type,
    entity_id,
    mentioned_user_id,
    mentioned_by_user_id
) VALUES (
    sqlc.arg(entity_type),
    sqlc.arg(entity_id),
    sqlc.arg(mentioned_user_id),
    sqlc.arg(mentioned_by_user_id)
) ON CONFLICT (entity_type, entity_id, mentioned_user_id) DO NOTHING
RETURNING *;

-- name: GetMentionedUsersByEntity :many
SELECT m.id, m.entity_type, m.entity_id, m.mentioned_user_id, m.created_at,
       u.username, u.full_name, u.avatar_url
FROM mentions m
JOIN users u ON m.mentioned_user_id = u.id
WHERE m.entity_type = sqlc.arg(entity_type) AND m.entity_id = sqlc.arg(entity_id)
ORDER BY m.created_at ASC;

-- name: GetUserMentionsForEntity :many
SELECT m.id, m.entity_type, m.entity_id, m.created_at,
       u.username as mentioned_by_username
FROM mentions m
JOIN users u ON m.mentioned_by_user_id = u.id
WHERE m.mentioned_user_id = sqlc.arg(user_id)
ORDER BY m.created_at DESC
LIMIT sqlc.arg(lim) OFFSET sqlc.arg(off);

-- name: DeleteMentionsByEntity :exec
DELETE FROM mentions
WHERE entity_type = $1 AND entity_id = $2;

-- name: CountUserMentions :one
SELECT COUNT(*) as count
FROM mentions
WHERE mentioned_user_id = $1
AND created_at > $2;