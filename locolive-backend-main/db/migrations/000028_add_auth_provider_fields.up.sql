ALTER TABLE users ADD COLUMN IF NOT EXISTS provider VARCHAR NOT NULL DEFAULT 'local';
ALTER TABLE users ADD COLUMN IF NOT EXISTS is_profile_complete BOOLEAN NOT NULL DEFAULT true;

CREATE INDEX IF NOT EXISTS idx_users_provider ON users(provider);
CREATE INDEX IF NOT EXISTS idx_users_profile_complete ON users(is_profile_complete);
