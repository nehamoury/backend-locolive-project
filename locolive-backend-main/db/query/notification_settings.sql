-- name: GetNotificationSettings :one
SELECT * FROM notification_settings WHERE user_id = $1;

-- name: UpsertNotificationSettings :one
INSERT INTO notification_settings (
    user_id, email_notifications, push_notifications, marketing_emails, activity_notifications, updated_at
) VALUES (
    $1, $2, $3, $4, $5, NOW()
) ON CONFLICT (user_id) DO UPDATE
SET 
    email_notifications = EXCLUDED.email_notifications,
    push_notifications = EXCLUDED.push_notifications,
    marketing_emails = EXCLUDED.marketing_emails,
    activity_notifications = EXCLUDED.activity_notifications,
    updated_at = NOW()
RETURNING *;
