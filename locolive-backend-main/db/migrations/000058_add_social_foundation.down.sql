



ALTER TABLE reels DROP COLUMN IF EXISTS engagement_score;
ALTER TABLE reels DROP COLUMN IF EXISTS watch_time_score;

ALTER TABLE posts DROP COLUMN IF EXISTS engagement_score;
ALTER TABLE posts DROP COLUMN IF EXISTS watch_time_score;

-- Note: Removing enum values is not supported natively in PostgreSQL without recreating the type.
-- We leave the new notification_type enum values alone in the down migration for safety.

ALTER TABLE notifications DROP COLUMN IF EXISTS read_at;
ALTER TABLE notifications DROP COLUMN IF EXISTS seen_at;

DROP TABLE IF EXISTS relationships;

DROP TYPE IF EXISTS relationship_type;
DROP TYPE IF EXISTS relationship_status;
