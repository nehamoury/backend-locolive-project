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
