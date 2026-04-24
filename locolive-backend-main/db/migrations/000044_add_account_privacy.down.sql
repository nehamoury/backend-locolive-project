DROP TABLE IF EXISTS privacy_logs;
ALTER TABLE users DROP COLUMN IF EXISTS privacy_updated_at;
ALTER TABLE users DROP COLUMN IF EXISTS is_private;