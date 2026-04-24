-- name: GetPrivacySettings :one
SELECT * FROM privacy_settings WHERE user_id = $1;

-- name: UpsertPrivacySettings :one
INSERT INTO privacy_settings (
    user_id, who_can_message, who_can_see_stories, show_location
) VALUES (
    $1, $2, $3, $4
) ON CONFLICT (user_id) DO UPDATE
SET 
    who_can_message = EXCLUDED.who_can_message,
    who_can_see_stories = EXCLUDED.who_can_see_stories,
    show_location = EXCLUDED.show_location,
    updated_at = NOW()
RETURNING *;

-- name: UpdateAccountPrivacy :one
UPDATE users
SET 
    is_private = $2,
    privacy_updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: LogPrivacyChange :one
INSERT INTO privacy_logs (
    user_id, old_value, new_value
) VALUES (
    $1, $2, $3
) RETURNING *;

