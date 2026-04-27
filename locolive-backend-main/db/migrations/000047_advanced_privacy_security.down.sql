-- Rollback: Advanced Privacy & Security System
DROP INDEX IF EXISTS idx_user_activity_logs_created_at;
DROP INDEX IF EXISTS idx_user_activity_logs_action;
DROP INDEX IF EXISTS idx_user_activity_logs_user_id;
DROP TABLE IF EXISTS user_activity_logs;

DROP INDEX IF EXISTS idx_users_deleted_at;
ALTER TABLE users DROP COLUMN IF EXISTS deleted_at;

DROP INDEX IF EXISTS idx_users_panic_mode;
ALTER TABLE users DROP COLUMN IF EXISTS panic_mode;
