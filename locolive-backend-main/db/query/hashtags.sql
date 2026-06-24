-- name: UpsertHashtag :one
INSERT INTO hashtags (name, slug, usage_count, reels_count, last_used_at)
VALUES ($1, $2, 1, 1, now())
ON CONFLICT (name) DO UPDATE 
SET usage_count = hashtags.usage_count + 1,
    reels_count = hashtags.reels_count + 1,
    last_used_at = now()
RETURNING *;

-- name: AddReelHashtag :exec
INSERT INTO reel_hashtags (reel_id, hashtag_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: AddPostHashtag :exec
INSERT INTO post_hashtags (post_id, hashtag_id)
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
    r.id, r.user_id, r.video_url, r.caption, r.is_ai_generated, r.location_name, r.geohash, r.category_id,
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

-- name: GetHashtagByName :one
SELECT * FROM hashtags WHERE name = $1 LIMIT 1;

-- name: ListPostsByHashtag :many
SELECT 
    p.id, p.user_id, p.media_url, p.media_type, p.caption, p.body_text, p.location_name, p.crop_settings, p.category_id,
    p.likes_count, p.comments_count, p.shares_count, p.created_at, p.updated_at,
    u.username, u.full_name, u.avatar_url,
    c.name as category_name, c.icon as category_icon,
    COALESCE((
        SELECT array_agg(h2.name)::text[] 
        FROM post_hashtags ph2 
        JOIN hashtags h2 ON ph2.hashtag_id = h2.id 
        WHERE ph2.post_id = p.id
    ), '{}')::text[] as hashtags,
    CASE WHEN p.geom IS NOT NULL THEN ST_Y(p.geom::geometry) ELSE NULL END as lat_out,
    CASE WHEN p.geom IS NOT NULL THEN ST_X(p.geom::geometry) ELSE NULL END as lng_out,
    EXISTS(SELECT 1 FROM post_likes pl WHERE pl.post_id = p.id AND pl.user_id = sqlc.arg(viewer_id)) as liked_by_viewer,
    EXISTS(SELECT 1 FROM post_saves ps WHERE ps.post_id = p.id AND ps.user_id = sqlc.arg(viewer_id)) as is_saved
FROM posts p
JOIN users u ON p.user_id = u.id
JOIN post_hashtags ph ON p.id = ph.post_id
JOIN hashtags h ON ph.hashtag_id = h.id
LEFT JOIN categories c ON p.category_id = c.id
WHERE h.name = sqlc.arg(hashtag_name)
ORDER BY p.created_at DESC
LIMIT sqlc.arg(lim) OFFSET sqlc.arg(off);
