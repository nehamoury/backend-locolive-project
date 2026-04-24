-- Revert username uniqueness system changes

-- Drop case-insensitive username index
DROP INDEX IF EXISTS idx_username_lower;

-- Drop username history table
DROP TABLE IF EXISTS username_history;

-- Note: We keep the username_original column in users table to avoid data loss
