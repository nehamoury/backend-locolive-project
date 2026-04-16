-- name: CreateLocation :one
INSERT INTO locations (
  user_id,
  geohash,
  geom,
  time_bucket,
  expires_at
) VALUES (
  @user_id, @geohash, ST_SetSRID(ST_MakePoint(@lng::float8, @lat::float8), 4326), @time_bucket, @expires_at
) RETURNING *;

-- name: DeleteExpiredLocations :exec
DELETE FROM locations
WHERE expires_at < now();

-- name: GetHeatmapData :many
SELECT 
  ST_X(ST_SnapToGrid(geom, 0.001)) as longitude,
  ST_Y(ST_SnapToGrid(geom, 0.001)) as latitude,
  COUNT(*) as weight
FROM locations
WHERE time_bucket > NOW() - INTERVAL '1 hour'
GROUP BY ST_SnapToGrid(geom, 0.001);

-- name: GetNearbyUsersFromDB :many
WITH recent_locations AS (
  SELECT DISTINCT ON (l.user_id)
    l.user_id,
    ST_X(l.geom) as longitude,
    ST_Y(l.geom) as latitude,
    ST_Distance(
      l.geom::geography,
      ST_SetSRID(ST_MakePoint(@lng::float8, @lat::float8), 4326)::geography
    ) / 1000.0 as distance_km
  FROM locations l
  WHERE l.expires_at > NOW()
    AND l.user_id != @exclude_user_id
    AND ST_DWithin(
      l.geom::geography,
      ST_SetSRID(ST_MakePoint(@lng::float8, @lat::float8), 4326)::geography,
      @radius_km * 1000
    )
  ORDER BY l.user_id, l.time_bucket DESC
)
SELECT 
  u.id as user_id,
  u.username,
  u.full_name,
  u.avatar_url,
  u.bio,
  u.is_ghost_mode,
  u.is_shadow_banned,
  u.last_active_at,
  r.longitude,
  r.latitude,
  r.distance_km
FROM recent_locations r
JOIN users u ON r.user_id = u.id
WHERE u.is_ghost_mode = false
  AND u.is_shadow_banned = false
ORDER BY r.distance_km ASC
LIMIT 50;
