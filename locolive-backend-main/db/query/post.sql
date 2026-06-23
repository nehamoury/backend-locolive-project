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
       EXISTS(SELECT 1 FROM post_likes pl WHERE pl.post_id = p.id AND pl.user_id = sqlc.arg(viewer_id)) as liked_by_viewer,
       EXISTS(SELECT 1 FROM post_saves ps WHERE ps.post_id = p.id AND ps.user_id = sqlc.arg(viewer_id)) as is_saved
FROM posts p
JOIN users u ON p.user_id = u.id
WHERE p.user_id = sqlc.arg(user_id)
ORDER BY p.created_at DESC
LIMIT sqlc.arg(lim) OFFSET sqlc.arg(off);

-- name: ListConnectionsPosts :many
-- Get posts from connections AND own posts AND nearby discovery (public posts)
SELECT DISTINCT ON (p.id) p.id, p.user_id, p.media_url, p.media_type, p.caption, p.body_text, p.location_name, p.crop_settings,
       p.likes_count, p.comments_count, p.shares_count, p.created_at, p.updated_at,
       u.username, u.full_name, u.avatar_url,
       CASE WHEN p.geom IS NOT NULL THEN ST_Y(p.geom::geometry) ELSE NULL END as lat_out,
       CASE WHEN p.geom IS NOT NULL THEN ST_X(p.geom::geometry) ELSE NULL END as lng_out,
       EXISTS(SELECT 1 FROM post_likes pl WHERE pl.post_id = p.id AND pl.user_id = sqlc.arg(viewer_id)) as liked_by_viewer,
       EXISTS(SELECT 1 FROM post_saves ps WHERE ps.post_id = p.id AND ps.user_id = sqlc.arg(viewer_id)) as is_saved
FROM posts p
JOIN users u ON p.user_id = u.id
LEFT JOIN connections c ON
    (c.requester_id = sqlc.arg(viewer_id) AND c.target_id = p.user_id) OR
    (c.target_id = sqlc.arg(viewer_id) AND c.requester_id = p.user_id)
WHERE (
    p.user_id = sqlc.arg(viewer_id) 
    OR c.status = 'accepted'
    OR (u.is_private = false AND u.is_shadow_banned = false) -- Discovery: Public users
)
AND NOT EXISTS (
    SELECT 1 FROM blocked_users bu
    WHERE (bu.blocker_id = sqlc.arg(viewer_id) AND bu.blocked_id = p.user_id)
       OR (bu.blocker_id = p.user_id AND bu.blocked_id = sqlc.arg(viewer_id))
)
ORDER BY p.id, p.created_at DESC
LIMIT sqlc.arg(lim) OFFSET sqlc.arg(off);

-- name: DeletePost :exec
DELETE FROM posts WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id);

-- name: AdminDeletePost :exec
DELETE FROM posts WHERE id = sqlc.arg(id);

-- name: LikePostAtomic :one
WITH inserted AS (
    INSERT INTO post_likes (post_id, user_id)
    VALUES (sqlc.arg(post_id), sqlc.arg(liker_id))
    ON CONFLICT (post_id, user_id) DO NOTHING
    RETURNING post_id
)
UPDATE posts p
SET likes_count = p.likes_count + 1
FROM inserted i
WHERE p.id = i.post_id
RETURNING p.likes_count;

-- name: UnlikePostAtomic :one
WITH deleted AS (
    DELETE FROM post_likes
    WHERE post_likes.post_id = sqlc.arg(post_id) AND post_likes.user_id = sqlc.arg(liker_id)
    RETURNING post_id
)
UPDATE posts p
SET likes_count = GREATEST(0, p.likes_count - 1)
FROM deleted d
WHERE p.id = d.post_id
RETURNING p.likes_count;

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

-- name: GetPost :one
SELECT * FROM posts WHERE id = sqlc.arg(id) LIMIT 1;

-- name: CountPostsByUserID :one
SELECT COUNT(*) FROM posts WHERE user_id = sqlc.arg(user_id);

-- name: ListLikedPostsByUserID :many
SELECT p.* FROM posts p
JOIN post_likes pl ON p.id = pl.post_id
WHERE pl.user_id = sqlc.arg(user_id)
ORDER BY pl.created_at DESC;

-- name: SavePostAtomic :one
WITH inserted AS (
    INSERT INTO post_saves (post_id, user_id)
    VALUES (sqlc.arg(post_id), sqlc.arg(user_id))
    ON CONFLICT (post_id, user_id) DO NOTHING
    RETURNING post_id
)
UPDATE posts p
SET saves_count = p.saves_count + 1
FROM inserted i
WHERE p.id = i.post_id
RETURNING p.saves_count;

-- name: UnsavePostAtomic :one
WITH deleted AS (
    DELETE FROM post_saves
    WHERE post_saves.post_id = sqlc.arg(post_id) AND post_saves.user_id = sqlc.arg(user_id)
    RETURNING post_id
)
UPDATE posts p
SET saves_count = GREATEST(0, p.saves_count - 1)
FROM deleted d
WHERE p.id = d.post_id
RETURNING p.saves_count;

-- name: ListSavedPosts :many
SELECT p.id, p.user_id, p.media_url, p.media_type, p.caption, p.body_text, p.location_name, p.crop_settings,
       p.likes_count, p.comments_count, p.shares_count, p.created_at, p.updated_at,
       u.username, u.full_name, u.avatar_url,
       TRUE as is_saved,
       EXISTS(SELECT 1 FROM post_likes pl WHERE pl.post_id = p.id AND pl.user_id = sqlc.arg(viewer_id)) as liked_by_viewer
FROM posts p
JOIN users u ON p.user_id = u.id
JOIN post_saves ps ON p.id = ps.post_id
WHERE ps.user_id = sqlc.arg(viewer_id)
ORDER BY ps.created_at DESC
LIMIT sqlc.arg(lim) OFFSET sqlc.arg(off);

-- name: CountSavedPosts :one
SELECT COUNT(*) FROM post_saves WHERE user_id = sqlc.arg(user_id);

-- name: ListAllPostsAdmin :many
SELECT 
    p.id, p.user_id, p.media_url, p.media_type, p.caption, p.body_text, p.location_name,
    p.likes_count, p.comments_count, p.shares_count, p.created_at, p.updated_at,
    u.username, u.avatar_url
FROM posts p
JOIN users u ON p.user_id = u.id
ORDER BY p.created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountAllPostsAdmin :one
SELECT COUNT(*) FROM posts;

-- name: UpdatePost :one
UPDATE posts SET caption = sqlc.arg(caption), updated_at = now() WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id) RETURNING *;
