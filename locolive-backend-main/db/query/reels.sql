-- name: CreateReel :one
INSERT INTO reels (
    user_id, video_url, caption, is_ai_generated, location_name, geohash, geom
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
) RETURNING id, user_id, video_url, caption, is_ai_generated, location_name, geohash, 
    COALESCE(ST_Y(geom::geometry)::float8, 0.0)::float8 AS lat, COALESCE(ST_X(geom::geometry)::float8, 0.0)::float8 AS lng,
    likes_count, comments_count, shares_count, saves_count, created_at, updated_at;

-- name: GetReel :one
SELECT id, user_id, video_url, caption, is_ai_generated, location_name, geohash, 
    COALESCE(ST_Y(geom::geometry)::float8, 0.0)::float8 AS lat, COALESCE(ST_X(geom::geometry)::float8, 0.0)::float8 AS lng,
    likes_count, comments_count, shares_count, saves_count, created_at, updated_at 
FROM reels WHERE id = $1 LIMIT 1;

-- name: ListReelsFeed :many
SELECT 
    r.id, r.user_id, r.video_url, r.caption, r.is_ai_generated, r.location_name, r.geohash,
    COALESCE(ST_Y(r.geom::geometry)::float8, 0.0)::float8 AS lat, COALESCE(ST_X(r.geom::geometry)::float8, 0.0)::float8 AS lng,
    r.likes_count, r.comments_count, r.shares_count, r.saves_count, r.created_at, r.updated_at,
    u.username,
    u.avatar_url,
    EXISTS (SELECT 1 FROM reel_likes rl WHERE rl.reel_id = r.id AND rl.user_id = $1) AS is_liked,
    EXISTS (SELECT 1 FROM reel_saves rs WHERE rs.reel_id = r.id AND rs.user_id = $1) AS is_saved,
    COALESCE((SELECT status FROM connections c WHERE (c.requester_id = $1 AND c.target_id = r.user_id) OR (c.requester_id = r.user_id AND c.target_id = $1) LIMIT 1)::text, 'none') AS connection_status
FROM reels r
JOIN users u ON r.user_id = u.id
ORDER BY r.created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListNearbyReels :many
SELECT 
    r.id, r.user_id, r.video_url, r.caption, r.is_ai_generated, r.location_name, r.geohash,
    COALESCE(ST_Y(r.geom::geometry)::float8, 0.0)::float8 AS lat, COALESCE(ST_X(r.geom::geometry)::float8, 0.0)::float8 AS lng,
    r.likes_count, r.comments_count, r.shares_count, r.saves_count, r.created_at, r.updated_at,
    u.username,
    u.avatar_url,
    ST_Distance(r.geom, ST_SetSRID(ST_MakePoint(sqlc.arg(lng)::float, sqlc.arg(lat)::float), 4326)::geography) AS distance_meters,
    EXISTS (SELECT 1 FROM reel_likes rl WHERE rl.reel_id = r.id AND rl.user_id = sqlc.arg(viewer_id)) AS is_liked,
    EXISTS (SELECT 1 FROM reel_saves rs WHERE rs.reel_id = r.id AND rs.user_id = sqlc.arg(viewer_id)) AS is_saved,
    COALESCE((SELECT status FROM connections c WHERE (c.requester_id = sqlc.arg(viewer_id) AND c.target_id = r.user_id) OR (c.requester_id = r.user_id AND c.target_id = sqlc.arg(viewer_id)) LIMIT 1)::text, 'none') AS connection_status
FROM reels r
JOIN users u ON r.user_id = u.id
WHERE ST_DWithin(r.geom, ST_SetSRID(ST_MakePoint(sqlc.arg(lng)::float, sqlc.arg(lat)::float), 4326)::geography, sqlc.arg(radius)::float)
ORDER BY distance_meters ASC
LIMIT $1 OFFSET $2;

-- name: ListUserReels :many
SELECT 
    r.id, r.user_id, r.video_url, r.caption, r.is_ai_generated, r.location_name, r.geohash,
    COALESCE(ST_Y(r.geom::geometry)::float8, 0.0)::float8 AS lat, COALESCE(ST_X(r.geom::geometry)::float8, 0.0)::float8 AS lng,
    r.likes_count, r.comments_count, r.shares_count, r.saves_count, r.created_at, r.updated_at,
    u.username,
    u.avatar_url,
    EXISTS (SELECT 1 FROM reel_likes rl WHERE rl.reel_id = r.id AND rl.user_id = sqlc.arg(viewer_id)) AS is_liked,
    EXISTS (SELECT 1 FROM reel_saves rs WHERE rs.reel_id = r.id AND rs.user_id = sqlc.arg(viewer_id)) AS is_saved,
    COALESCE((SELECT status FROM connections c WHERE (c.requester_id = sqlc.arg(viewer_id) AND c.target_id = r.user_id) OR (c.requester_id = r.user_id AND c.target_id = sqlc.arg(viewer_id)) LIMIT 1)::text, 'none') AS connection_status
FROM reels r
JOIN users u ON r.user_id = u.id
WHERE r.user_id = sqlc.arg(user_id)
ORDER BY r.created_at DESC
LIMIT $1 OFFSET $2;

-- name: LikeReelAtomic :one
WITH inserted AS (
    INSERT INTO reel_likes (reel_id, user_id)
    VALUES (sqlc.arg(reel_id), sqlc.arg(liker_id))
    ON CONFLICT (reel_id, user_id) DO NOTHING
    RETURNING reel_id
)
UPDATE reels r
SET likes_count = r.likes_count + 1
FROM inserted i
WHERE r.id = i.reel_id
RETURNING r.likes_count;

-- name: UnlikeReelAtomic :one
WITH deleted AS (
    DELETE FROM reel_likes 
    WHERE reel_likes.reel_id = sqlc.arg(reel_id) AND reel_likes.user_id = sqlc.arg(liker_id)
    RETURNING reel_id
)
UPDATE reels r
SET likes_count = GREATEST(0, r.likes_count - 1)
FROM deleted d
WHERE r.id = d.reel_id
RETURNING r.likes_count;

-- name: CreateReelComment :one
INSERT INTO reel_comments (reel_id, user_id, content, is_flagged)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: ListReelComments :many
SELECT 
    rc.*,
    u.username,
    u.avatar_url
FROM reel_comments rc
JOIN users u ON rc.user_id = u.id
WHERE rc.reel_id = $1
ORDER BY rc.created_at DESC;

-- name: IncrementReelComments :exec
UPDATE reels SET comments_count = comments_count + 1 WHERE id = $1;

-- name: SaveReelAtomic :one
WITH inserted AS (
    INSERT INTO reel_saves (reel_id, user_id)
    VALUES (sqlc.arg(reel_id), sqlc.arg(saver_id))
    ON CONFLICT (reel_id, user_id) DO NOTHING
    RETURNING reel_id
)
UPDATE reels r
SET saves_count = r.saves_count + 1
FROM inserted i
WHERE r.id = i.reel_id
RETURNING r.saves_count;

-- name: UnsaveReelAtomic :one
WITH deleted AS (
    DELETE FROM reel_saves
    WHERE reel_saves.reel_id = sqlc.arg(reel_id) AND reel_saves.user_id = sqlc.arg(saver_id)
    RETURNING reel_id
)
UPDATE reels r
SET saves_count = GREATEST(0, r.saves_count - 1)
FROM deleted d
WHERE r.id = d.reel_id
RETURNING r.saves_count;

-- name: IncrementReelShares :exec
UPDATE reels SET shares_count = shares_count + 1 WHERE id = $1;

-- name: GetTotalReelsCountToday :one
SELECT COUNT(*) FROM reels WHERE created_at >= CURRENT_DATE;

-- name: GetReelComment :one
SELECT * FROM reel_comments WHERE id = $1 LIMIT 1;

-- name: DeleteReelComment :one
DELETE FROM reel_comments WHERE id = $1 AND user_id = $2
RETURNING reel_id;

-- name: AdminDeleteReelComment :one
DELETE FROM reel_comments WHERE id = $1
RETURNING reel_id;

-- name: DecrementReelComments :exec
UPDATE reels SET comments_count = GREATEST(0, comments_count - 1) WHERE id = $1;

-- name: DeleteReel :exec
DELETE FROM reels WHERE id = $1 AND user_id = $2;

-- name: AdminDeleteReel :exec
DELETE FROM reels WHERE id = $1;

-- name: ListSavedReels :many
SELECT 
    r.id, r.user_id, r.video_url, r.caption, r.is_ai_generated, r.location_name, r.geohash,
    COALESCE(ST_Y(r.geom::geometry)::float8, 0.0)::float8 AS lat, COALESCE(ST_X(r.geom::geometry)::float8, 0.0)::float8 AS lng,
    r.likes_count, r.comments_count, r.shares_count, r.saves_count, r.created_at, r.updated_at,
    u.username,
    u.avatar_url,
    TRUE AS is_saved,
    EXISTS (SELECT 1 FROM reel_likes rl WHERE rl.reel_id = r.id AND rl.user_id = $1) AS is_liked,
    COALESCE((SELECT status FROM connections c WHERE (c.requester_id = $1 AND c.target_id = r.user_id) OR (c.requester_id = r.user_id AND c.target_id = $1) LIMIT 1)::text, 'none') AS connection_status
FROM reels r
JOIN reel_saves rs ON r.id = rs.reel_id
JOIN users u ON r.user_id = u.id
WHERE rs.user_id = $1
ORDER BY rs.created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountReelsByUserID :one
SELECT COUNT(*) FROM reels WHERE user_id = sqlc.arg(user_id);

-- name: ListLikedReelsByUserID :many
SELECT r.* FROM reels r
JOIN reel_likes rl ON r.id = rl.reel_id
WHERE rl.user_id = sqlc.arg(user_id)
ORDER BY rl.created_at DESC;

-- name: CountSavedReels :one
SELECT COUNT(*) FROM reel_saves WHERE user_id = sqlc.arg(user_id);

-- name: ListAllReelsAdmin :many
SELECT 
    r.id, r.user_id, r.video_url, r.caption, r.is_ai_generated, r.location_name, r.geohash,
    r.likes_count, r.comments_count, r.shares_count, r.saves_count, r.created_at, r.updated_at,
    u.username, u.avatar_url
FROM reels r
JOIN users u ON r.user_id = u.id
ORDER BY r.created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountAllReelsAdmin :one
SELECT COUNT(*) FROM reels;

-- name: UpdateReel :one
UPDATE reels SET caption = sqlc.arg(caption), updated_at = now() WHERE id = sqlc.arg(id) AND user_id = sqlc.arg(user_id) RETURNING *;
