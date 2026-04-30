-- name: CreatePost :one
INSERT INTO posts (user_id, media_url, media_type, caption, body_text, location_name, geohash, geom, crop_settings)
VALUES (
    sqlc.arg(user_id), sqlc.arg(media_url), sqlc.arg(media_type),
    sqlc.narg(caption), sqlc.narg(body_text), sqlc.narg(location_name), sqlc.narg(geohash),

    CASE WHEN sqlc.arg(has_location)::boolean
         THEN ST_SetSRID(ST_MakePoint(sqlc.arg(lng)::float8, sqlc.arg(lat)::float8), 4326)
         ELSE NULL END,
    sqlc.narg(crop_settings)
)
RETURNING *,
    CASE WHEN geom IS NOT NULL THEN ST_Y(geom::geometry) ELSE NULL END as lat_out,
    CASE WHEN geom IS NOT NULL THEN ST_X(geom::geometry) ELSE NULL END as lng_out;

-- name: ListPostsByUserID :many
SELECT p.id, p.user_id, p.media_url, p.media_type, p.caption, p.body_text, p.location_name, p.crop_settings,
       p.likes_count, p.comments_count, p.shares_count, p.created_at, p.updated_at,

       u.username, u.full_name, u.avatar_url,
       CASE WHEN p.geom IS NOT NULL THEN ST_Y(p.geom::geometry) ELSE NULL END as lat_out,
       CASE WHEN p.geom IS NOT NULL THEN ST_X(p.geom::geometry) ELSE NULL END as lng_out,
       EXISTS(SELECT 1 FROM post_likes pl WHERE pl.post_id = p.id AND pl.user_id = sqlc.arg(viewer_id)) as liked_by_viewer
FROM posts p
JOIN users u ON p.user_id = u.id
WHERE p.user_id = sqlc.arg(user_id)
ORDER BY p.created_at DESC
LIMIT sqlc.arg(lim) OFFSET sqlc.arg(off);

-- name: ListConnectionsPosts :many
-- Get posts from connections AND own posts
SELECT p.id, p.user_id, p.media_url, p.media_type, p.caption, p.body_text, p.location_name, p.crop_settings,
       p.likes_count, p.comments_count, p.shares_count, p.created_at, p.updated_at,

       u.username, u.full_name, u.avatar_url,
       CASE WHEN p.geom IS NOT NULL THEN ST_Y(p.geom::geometry) ELSE NULL END as lat_out,
       CASE WHEN p.geom IS NOT NULL THEN ST_X(p.geom::geometry) ELSE NULL END as lng_out,
       EXISTS(SELECT 1 FROM post_likes pl WHERE pl.post_id = p.id AND pl.user_id = sqlc.arg(viewer_id)) as liked_by_viewer
FROM posts p
JOIN users u ON p.user_id = u.id
LEFT JOIN connections c ON
    (c.requester_id = sqlc.arg(viewer_id) AND c.target_id = p.user_id) OR
    (c.target_id = sqlc.arg(viewer_id) AND c.requester_id = p.user_id)
WHERE (p.user_id = sqlc.arg(viewer_id) OR (c.status = 'accepted' AND u.is_shadow_banned = false))
    AND NOT EXISTS (
        SELECT 1 FROM blocked_users bu
        WHERE (bu.blocker_id = sqlc.arg(viewer_id) AND bu.blocked_id = p.user_id)
           OR (bu.blocker_id = p.user_id AND bu.blocked_id = sqlc.arg(viewer_id))
    )
ORDER BY p.created_at DESC
LIMIT sqlc.arg(lim) OFFSET sqlc.arg(off);

-- name: DeletePost :exec
DELETE FROM posts WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id);

-- name: LikePost :one
INSERT INTO post_likes (post_id, user_id) VALUES (sqlc.arg(post_id), sqlc.arg(user_id))
ON CONFLICT (post_id, user_id) DO NOTHING
RETURNING *;

-- name: UnlikePost :exec
DELETE FROM post_likes WHERE post_id = sqlc.arg(post_id) AND user_id = sqlc.arg(user_id);

-- name: IncrementPostLikes :exec
UPDATE posts SET likes_count = likes_count + 1 WHERE id = sqlc.arg(id);

-- name: DecrementPostLikes :exec
UPDATE posts SET likes_count = GREATEST(0, likes_count - 1) WHERE id = sqlc.arg(id);

-- name: IncrementPostShares :exec
UPDATE posts SET shares_count = shares_count + 1 WHERE id = sqlc.arg(id);

-- name: CreatePostComment :one
INSERT INTO post_comments (post_id, user_id, content, is_flagged)
VALUES (sqlc.arg(post_id), sqlc.arg(user_id), sqlc.arg(content), sqlc.arg(is_flagged))
RETURNING *;

-- name: ListPostComments :many
SELECT pc.id, pc.post_id, pc.user_id, pc.content, pc.created_at,
       u.username, u.full_name, u.avatar_url
FROM post_comments pc
JOIN users u ON pc.user_id = u.id
WHERE pc.post_id = sqlc.arg(post_id)
ORDER BY pc.created_at ASC
LIMIT 20;

-- name: DeletePostComment :one
DELETE FROM post_comments WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id)
RETURNING post_id;


-- name: AdminDeletePostComment :one
DELETE FROM post_comments WHERE id = sqlc.arg(id)
RETURNING post_id;


-- name: GetPostComment :one
SELECT * FROM post_comments WHERE id = sqlc.arg(id) LIMIT 1;

-- name: IncrementPostComments :exec
UPDATE posts SET comments_count = comments_count + 1 WHERE id = sqlc.arg(id);

-- name: DecrementPostComments :exec
UPDATE posts SET comments_count = GREATEST(0, comments_count - 1) WHERE id = sqlc.arg(id);

