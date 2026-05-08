-- Comment Mentions System (Future-friendly generic design)
-- Supports: post_comments, reel_comments, and expandable to stories/chats/captions later

-- Drop existing comment_mentions table if exists (clean rebuild)
DROP TABLE IF EXISTS mentions CASCADE;

-- Main mentions table with entity_type for future expansion
CREATE TABLE mentions (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    entity_type varchar(50) NOT NULL CHECK (entity_type IN ('post_comment', 'reel_comment', 'story', 'caption', 'chat_message')),
    entity_id uuid NOT NULL,
    mentioned_user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    mentioned_by_user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE(entity_type, entity_id, mentioned_user_id)
);

-- Indexes for common queries
CREATE INDEX idx_mentions_entity ON mentions(entity_type, entity_id);
CREATE INDEX idx_mentions_user ON mentions(mentioned_user_id, created_at DESC);
CREATE INDEX idx_mentions_by_user ON mentions(mentioned_by_user_id, created_at DESC);

-- Add is_flagged column to post_comments (added in 000036)
ALTER TABLE post_comments ADD COLUMN IF NOT EXISTS is_flagged boolean NOT NULL DEFAULT false;

-- Add is_flagged column to reel_comments
ALTER TABLE reel_comments ADD COLUMN IF NOT EXISTS is_flagged boolean NOT NULL DEFAULT false;