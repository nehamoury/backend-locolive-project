-- Create activity_logs table
CREATE TABLE activity_logs (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    action_type varchar NOT NULL, -- 'comment', 'report', 'like', 'share', 'reel_upload', 'post_upload'
    target_id uuid,               -- uuid of post/reel/story/user
    target_type varchar,           -- 'post', 'reel', 'story', 'user'
    details jsonb,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_activity_logs_user_id ON activity_logs(user_id);
CREATE INDEX idx_activity_logs_created_at ON activity_logs(created_at);

-- Add trust score to users
ALTER TABLE users ADD COLUMN trust_score int NOT NULL DEFAULT 100;

-- Add flagged status to comments
ALTER TABLE post_comments ADD COLUMN is_flagged boolean NOT NULL DEFAULT false;
ALTER TABLE reel_comments ADD COLUMN is_flagged boolean NOT NULL DEFAULT false;

-- Enhance reports system
ALTER TABLE reports ADD COLUMN target_id uuid;
ALTER TABLE reports ADD COLUMN target_type varchar;
ALTER TABLE reports ADD COLUMN priority_score int NOT NULL DEFAULT 1;

-- Migrate existing report data
UPDATE reports SET 
    target_id = COALESCE(target_user_id, target_story_id),
    target_type = CASE 
        WHEN target_user_id IS NOT NULL THEN 'user'
        WHEN target_story_id IS NOT NULL THEN 'story'
        ELSE 'unknown'
    END;

-- Add indexes for activity sorting
CREATE INDEX idx_reports_priority ON reports(priority_score DESC);
CREATE INDEX idx_reports_target ON reports(target_type, target_id);
