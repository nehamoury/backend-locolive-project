DROP TABLE IF EXISTS activity_logs;

ALTER TABLE users DROP COLUMN IF EXISTS trust_score;

ALTER TABLE post_comments DROP COLUMN IF EXISTS is_flagged;
ALTER TABLE reel_comments DROP COLUMN IF EXISTS is_flagged;

ALTER TABLE reports DROP COLUMN IF EXISTS target_id;
ALTER TABLE reports DROP COLUMN IF EXISTS target_type;
ALTER TABLE reports DROP COLUMN IF EXISTS priority_score;

DROP INDEX IF EXISTS idx_reports_priority;
DROP INDEX IF EXISTS idx_reports_target;
