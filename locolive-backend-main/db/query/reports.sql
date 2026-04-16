-- name: CreateReport :one
INSERT INTO reports (
  reporter_id,
  target_id,
  target_type,
  reason,
  description,
  priority_score
) VALUES (
  $1, $2, $3, $4, $5, $6
) RETURNING *;

-- Admin: List all reports
-- name: ListReports :many
SELECT r.*, 
  u1.username as reporter_username,
  u1.avatar_url as reporter_avatar,
  COALESCE(u2.username, 'deleted') as target_username
FROM reports r
LEFT JOIN users u1 ON r.reporter_id = u1.id
LEFT JOIN users u2 ON r.target_id = u2.id AND r.target_type = 'user'
WHERE is_resolved = $1
ORDER BY r.priority_score DESC, r.created_at DESC
LIMIT $2 OFFSET $3;

-- Admin: Resolve report
-- name: ResolveReport :one
UPDATE reports
SET is_resolved = true
WHERE id = $1
RETURNING *;

-- name: IncrementReportPriority :exec
UPDATE reports
SET priority_score = priority_score + 1
WHERE target_id = $1 AND is_resolved = false;
