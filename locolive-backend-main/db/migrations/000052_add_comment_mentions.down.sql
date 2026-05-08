-- Rollback Comment Mentions System
DROP TABLE IF EXISTS mentions CASCADE;

-- Note: We don't drop is_flagged columns as they may be used by moderation