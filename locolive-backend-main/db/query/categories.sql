-- name: CreateCategory :one
INSERT INTO categories (
    name,
    slug,
    icon,
    color,
    is_active
) VALUES (
    $1, $2, $3, $4, $5
) RETURNING *;

-- name: GetCategory :one
SELECT * FROM categories
WHERE id = $1 LIMIT 1;

-- name: GetCategoryBySlug :one
SELECT * FROM categories
WHERE slug = $1 LIMIT 1;

-- name: ListCategories :many
SELECT * FROM categories
WHERE is_active = true
ORDER BY name ASC;

-- name: UpdateCategory :one
UPDATE categories
SET
    name = COALESCE(sqlc.narg(name), name),
    slug = COALESCE(sqlc.narg(slug), slug),
    icon = COALESCE(sqlc.narg(icon), icon),
    color = COALESCE(sqlc.narg(color), color),
    is_active = COALESCE(sqlc.narg(is_active), is_active)
WHERE id = sqlc.arg(id)
RETURNING *;

-- name: DeleteCategory :exec
DELETE FROM categories
WHERE id = $1;

-- name: IncrementCategoryStats :exec
INSERT INTO category_stats (
    category_id, posts_count, reels_count, stories_count
) VALUES (
    $1, $2, $3, $4
) ON CONFLICT (category_id) DO UPDATE
SET
    posts_count = category_stats.posts_count + EXCLUDED.posts_count,
    reels_count = category_stats.reels_count + EXCLUDED.reels_count,
    stories_count = category_stats.stories_count + EXCLUDED.stories_count,
    updated_at = now();

-- name: ListTrendingCategories :many
SELECT c.*, s.posts_count, s.reels_count, s.stories_count
FROM categories c
JOIN category_stats s ON c.id = s.category_id
WHERE c.is_active = true
ORDER BY (s.posts_count + s.reels_count + s.stories_count) DESC
LIMIT $1 OFFSET $2;

-- name: ListCategoryPosts :many
SELECT p.id, p.user_id, p.media_url, p.media_type, p.caption, p.body_text, p.location_name, p.crop_settings, p.category_id,
       p.likes_count, p.comments_count, p.shares_count, p.created_at, p.updated_at,
       u.username, u.full_name, u.avatar_url,
       CASE WHEN p.geom IS NOT NULL THEN ST_Y(p.geom::geometry) ELSE NULL END as lat_out,
       CASE WHEN p.geom IS NOT NULL THEN ST_X(p.geom::geometry) ELSE NULL END as lng_out,
       EXISTS(SELECT 1 FROM post_likes pl WHERE pl.post_id = p.id AND pl.user_id = sqlc.arg(viewer_id)) as liked_by_viewer,
       EXISTS(SELECT 1 FROM post_saves ps WHERE ps.post_id = p.id AND ps.user_id = sqlc.arg(viewer_id)) as is_saved
FROM posts p
JOIN users u ON p.user_id = u.id
WHERE p.category_id = sqlc.arg(category_id)
  AND u.is_shadow_banned = false
ORDER BY p.created_at DESC
LIMIT sqlc.arg(lim) OFFSET sqlc.arg(off);

-- name: ListCategoryReels :many
SELECT r.id, r.user_id, r.video_url, r.caption, r.is_ai_generated, r.location_name, r.geohash, r.category_id,
       COALESCE(ST_Y(r.geom::geometry)::float8, 0.0)::float8 AS lat, COALESCE(ST_X(r.geom::geometry)::float8, 0.0)::float8 AS lng,
       r.likes_count, r.comments_count, r.shares_count, r.saves_count, r.created_at, r.updated_at,
       u.username, u.avatar_url,
       EXISTS (SELECT 1 FROM reel_likes rl WHERE rl.reel_id = r.id AND rl.user_id = sqlc.arg(viewer_id)) AS is_liked,
       EXISTS (SELECT 1 FROM reel_saves rs WHERE rs.reel_id = r.id AND rs.user_id = sqlc.arg(viewer_id)) AS is_saved,
       COALESCE((SELECT status FROM connections c WHERE (c.requester_id = sqlc.arg(viewer_id) AND c.target_id = r.user_id) OR (c.requester_id = r.user_id AND c.target_id = sqlc.arg(viewer_id)) LIMIT 1)::text, 'none') AS connection_status
FROM reels r
JOIN users u ON r.user_id = u.id
WHERE r.category_id = sqlc.arg(category_id)
  AND u.is_shadow_banned = false
ORDER BY r.created_at DESC
LIMIT $1 OFFSET $2;

-- name: ListCategoryCreators :many
SELECT u.id, u.username, u.full_name, u.avatar_url, u.is_verified, u.bio,
       COUNT(DISTINCT p.id) + COUNT(DISTINCT r.id) AS content_count
FROM users u
LEFT JOIN posts p ON p.user_id = u.id AND p.category_id = sqlc.arg(category_id)
LEFT JOIN reels r ON r.user_id = u.id AND r.category_id = sqlc.arg(category_id)
WHERE u.is_shadow_banned = false
GROUP BY u.id, u.username, u.full_name, u.avatar_url, u.is_verified, u.bio
HAVING COUNT(DISTINCT p.id) + COUNT(DISTINCT r.id) > 0
ORDER BY content_count DESC
LIMIT $1 OFFSET $2;
