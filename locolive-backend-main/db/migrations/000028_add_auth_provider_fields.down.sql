DROP INDEX IF EXISTS idx_users_profile_complete;
DROP INDEX IF EXISTS idx_users_provider;
ALTER TABLE users DROP COLUMN IF EXISTS is_profile_complete;
ALTER TABLE users DROP COLUMN IF EXISTS provider;
