-- name: GetUserPreferences :one
SELECT * FROM user_preferences WHERE user_id = $1;

-- name: UpsertUserPreferences :one
INSERT INTO user_preferences (
    user_id, theme, language, content_filter_enabled, updated_at
) VALUES (
    $1, $2, $3, $4, NOW()
) ON CONFLICT (user_id) DO UPDATE
SET 
    theme = EXCLUDED.theme,
    language = EXCLUDED.language,
    content_filter_enabled = EXCLUDED.content_filter_enabled,
    updated_at = NOW()
RETURNING *;
