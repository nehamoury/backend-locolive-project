-- Advanced Privacy & Security System
-- Adds: panic_mode, soft deletes, audit logging

-- 1. Add panic_mode to users
ALTER TABLE users ADD COLUMN IF NOT EXISTS panic_mode BOOLEAN NOT NULL DEFAULT false;

-- 2. Add soft delete support
ALTER TABLE users ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ DEFAULT NULL;

-- 3. Create audit/activity log table for tracking critical user actions
CREATE TABLE IF NOT EXISTS user_activity_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    action VARCHAR NOT NULL,
    details JSONB,
    ip_address VARCHAR,
    user_agent VARCHAR,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 4. Performance indexes
CREATE INDEX IF NOT EXISTS idx_users_panic_mode ON users(panic_mode) WHERE panic_mode = true;
CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON users(deleted_at) WHERE deleted_at IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_user_activity_logs_user_id ON user_activity_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_user_activity_logs_action ON user_activity_logs(action);
CREATE INDEX IF NOT EXISTS idx_user_activity_logs_created_at ON user_activity_logs(created_at DESC);
