ALTER TABLE users ADD COLUMN IF NOT EXISTS password_reset_token VARCHAR;
ALTER TABLE users ADD COLUMN IF NOT EXISTS password_reset_expires_at TIMESTAMPTZ;
DROP TABLE IF EXISTS password_resets;
