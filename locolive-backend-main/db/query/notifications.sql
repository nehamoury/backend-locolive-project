-- name: CreateNotification :one
INSERT INTO notifications (
  user_id,
  type,
  title,
  message,
  related_user_id,
  related_story_id,
  related_crossing_id
) VALUES (
  $1, $2, $3, $4, $5, $6, $7
) RETURNING *;

-- name: ListNotifications :many
SELECT * FROM notifications
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: DeleteConnectionRequestNotifications :exec
DELETE FROM notifications
WHERE user_id = $1 
  AND type = 'connection_request' 
  AND related_user_id = $2;

-- name: MarkNotificationAsRead :one
UPDATE notifications
SET is_read = true
WHERE id = $1 AND user_id = $2
RETURNING *;

-- name: MarkAllNotificationsAsRead :exec
UPDATE notifications
SET is_read = true
WHERE user_id = $1 AND is_read = false;

-- name: CountUnreadNotifications :one
SELECT COUNT(*) FROM notifications
WHERE user_id = $1 AND is_read = false;

-- name: DeleteOldNotifications :exec
-- Delete notifications older than 30 days
DELETE FROM notifications
WHERE created_at < NOW() - INTERVAL '30 days';

-- Admin: List all notifications (for admin panel)
-- name: ListNotificationsAdmin :many
SELECT * FROM notifications
WHERE type = 'system_announcement'
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- Admin: Count notifications
-- name: CountNotificationsAdmin :one
SELECT COUNT(*) FROM notifications
WHERE type = 'system_announcement';
