DROP INDEX IF EXISTS idx_users_username_trgm;
-- We usually don't drop the extension in down migration if other things might use it
-- but since we just added it for this, we can consider it.
-- However, safe approach is just dropping the index.
