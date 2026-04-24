-- Username Uniqueness System Queries

-- name: GetUserByUsernameCaseInsensitive :one
-- Get user by username (case-insensitive match)
SELECT * FROM users
WHERE LOWER(username) = LOWER($1) LIMIT 1;

-- name: CheckUsernameExists :one
-- Check if a username exists (case-insensitive)
SELECT EXISTS(SELECT 1 FROM users WHERE LOWER(username) = LOWER($1)) as exists;

-- name: GetReservedUsernames :many
-- Get all reserved usernames
SELECT username FROM reserved_usernames;

-- name: IsUsernameReserved :one
-- Check if a username is reserved
SELECT EXISTS(SELECT 1 FROM reserved_usernames WHERE LOWER(username) = LOWER($1)) as reserved;

-- name: AddReservedUsername :exec
-- Add a new reserved username (admin only)
INSERT INTO reserved_usernames (username, reason) VALUES ($1, $2);

-- name: RemoveReservedUsername :exec
-- Remove a reserved username (admin only)
DELETE FROM reserved_usernames WHERE LOWER(username) = LOWER($1);

-- name: RecordUsernameChange :one
-- Record a username change in history
INSERT INTO username_history (user_id, old_username, new_username, changed_by) 
VALUES ($1, $2, $3, $4) 
RETURNING *;

-- name: GetUsernameHistory :many
-- Get username change history for a user
SELECT * FROM username_history 
WHERE user_id = $1 
ORDER BY changed_at DESC;

-- name: GetUserByPreviousUsername :one
-- Find user by a previous username (for redirects/mentions)
SELECT u.* FROM users u
JOIN username_history h ON u.id = h.user_id
WHERE LOWER(h.old_username) = LOWER($1)
ORDER BY h.changed_at DESC
LIMIT 1;

-- name: UpdateUsername :one
-- Update user username with history tracking
-- Note: This should be called within a transaction that also records history
UPDATE users 
SET username = $2, username_normalized = LOWER($2)
WHERE id = $1
RETURNING *;

-- name: FindSimilarUsernames :many
-- Find usernames similar to the given pattern (for suggestions)
-- Uses trigram similarity if available, otherwise simple LIKE
SELECT username FROM users
WHERE username ILIKE '%' || $1 || '%'
LIMIT 10;
