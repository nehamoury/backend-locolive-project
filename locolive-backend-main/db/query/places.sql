-- name: SearchPlaces :many
SELECT * FROM places
WHERE name ILIKE '%' || sqlc.arg(query)::text || '%'
ORDER BY post_count DESC, name ASC
LIMIT sqlc.arg(lim) OFFSET sqlc.arg(off);

-- name: GetPlace :one
SELECT * FROM places WHERE id = $1 LIMIT 1;

-- name: GetPlaceBySlug :one
SELECT * FROM places WHERE slug = $1 LIMIT 1;

-- name: ListTrendingPlaces :many
SELECT * FROM places
ORDER BY post_count DESC, created_at DESC
LIMIT $1 OFFSET $2;

-- name: UpsertPlace :one
INSERT INTO places (name, slug, address, geom)
VALUES ($1, $2, $3, 
    CASE WHEN $4::bool THEN ST_SetSRID(ST_MakePoint($5::float8, $6::float8), 4326) ELSE NULL END)
ON CONFLICT (name) DO UPDATE SET post_count = places.post_count + 1
RETURNING *;

-- name: IncrementPlacePostCount :exec
UPDATE places SET post_count = post_count + 1 WHERE id = $1;
