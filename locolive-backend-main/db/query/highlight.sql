-- name: CreateHighlight :one
INSERT INTO highlight_groups (user_id, title, cover_url)
VALUES (@user_id, @title, @cover_url)
RETURNING *;

-- name: ListHighlightsByUserID :many
SELECT hg.*,
    (SELECT COUNT(*) FROM highlight_stories WHERE highlight_id = hg.id) as story_count
FROM highlight_groups hg
WHERE hg.user_id = @user_id
ORDER BY hg.created_at ASC;

-- name: GetHighlightDetails :many
-- Returns all archived stories belonging to a highlight
SELECT a.*, hs.added_at
FROM highlight_stories hs
JOIN archived_stories a ON hs.archived_story_id = a.id
WHERE hs.highlight_id = @highlight_id
ORDER BY hs.added_at ASC;

-- name: AddStoryToHighlight :one
INSERT INTO highlight_stories (highlight_id, archived_story_id)
VALUES (@highlight_id, @archived_story_id)
ON CONFLICT (highlight_id, archived_story_id) DO NOTHING
RETURNING *;

-- name: RemoveStoryFromHighlight :exec
DELETE FROM highlight_stories WHERE highlight_id = @highlight_id AND archived_story_id = @archived_story_id;

-- name: DeleteHighlight :exec
DELETE FROM highlight_groups WHERE id = @id AND user_id = @user_id;

-- name: UpdateHighlightCover :one
UPDATE highlight_groups SET cover_url = @cover_url, updated_at = now()
WHERE id = @id AND user_id = @user_id
RETURNING *;
