-- name: UpsertHashtag :one
INSERT INTO hashtags (name, usage_count, reels_count, last_used_at)
VALUES ($1, 1, 1, now())
ON CONFLICT (name) DO UPDATE 
SET usage_count = hashtags.usage_count + 1,
    reels_count = hashtags.reels_count + 1,
    last_used_at = now()
RETURNING *;

-- name: AddReelHashtag :exec
INSERT INTO reel_hashtags (reel_id, hashtag_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: SearchHashtags :many
SELECT * FROM hashtags
WHERE name ILIKE $1 || '%'
ORDER BY usage_count DESC, last_used_at DESC
LIMIT $2;

-- name: GetTrendingHashtags :many
SELECT * FROM hashtags
ORDER BY 
    (usage_count + 
     (CASE WHEN last_used_at >= now() - interval '24 hours' THEN 3 ELSE 0 END) +
     (CASE WHEN last_used_at >= now() - interval '7 days' THEN 2 ELSE 0 END)) DESC,
    last_used_at DESC
LIMIT $1;

-- name: ListReelsByHashtag :many
SELECT 
    r.id, r.user_id, r.video_url, r.caption, r.is_ai_generated, r.location_name, r.geohash,
    COALESCE(ST_Y(r.geom::geometry)::float8, 0.0)::float8 AS lat, COALESCE(ST_X(r.geom::geometry)::float8, 0.0)::float8 AS lng,
    r.likes_count, r.comments_count, r.shares_count, r.saves_count, r.created_at, r.updated_at,
    u.username, u.avatar_url,
    EXISTS (SELECT 1 FROM reel_likes rl WHERE rl.reel_id = r.id AND rl.user_id = sqlc.arg(viewer_id)) AS is_liked,
    EXISTS (SELECT 1 FROM reel_saves rs WHERE rs.reel_id = r.id AND rs.user_id = sqlc.arg(viewer_id)) AS is_saved,
    COALESCE((SELECT status FROM connections c WHERE (c.requester_id = sqlc.arg(viewer_id) AND c.target_id = r.user_id) OR (c.requester_id = r.user_id AND c.target_id = sqlc.arg(viewer_id)) LIMIT 1)::text, 'none') AS connection_status
FROM reels r
JOIN users u ON r.user_id = u.id
JOIN reel_hashtags rh ON r.id = rh.reel_id
JOIN hashtags h ON rh.hashtag_id = h.id
WHERE h.name = sqlc.arg(hashtag_name)
ORDER BY r.created_at DESC
LIMIT $1 OFFSET $2;
