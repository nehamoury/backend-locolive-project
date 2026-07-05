-- Relationships Table
CREATE TYPE relationship_status AS ENUM ('pending', 'active', 'blocked');
CREATE TYPE relationship_type AS ENUM ('follow', 'close_friend', 'business', 'favorite', 'mute', 'block');

CREATE TABLE relationships (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    target_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type relationship_type NOT NULL DEFAULT 'follow',
    status relationship_status NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(user_id, target_user_id, type)
);

CREATE INDEX idx_relationships_user_id ON relationships(user_id);
CREATE INDEX idx_relationships_target_user_id ON relationships(target_user_id);

-- Add read_at and seen_at to notifications
ALTER TABLE notifications ADD COLUMN read_at TIMESTAMPTZ;
ALTER TABLE notifications ADD COLUMN seen_at TIMESTAMPTZ;

-- Add new Notification Types using ALTER TYPE
ALTER TYPE notification_type ADD VALUE IF NOT EXISTS 'follow';
ALTER TYPE notification_type ADD VALUE IF NOT EXISTS 'like';
ALTER TYPE notification_type ADD VALUE IF NOT EXISTS 'comment';
ALTER TYPE notification_type ADD VALUE IF NOT EXISTS 'post_liked';
ALTER TYPE notification_type ADD VALUE IF NOT EXISTS 'reel_liked';
ALTER TYPE notification_type ADD VALUE IF NOT EXISTS 'post_commented';
ALTER TYPE notification_type ADD VALUE IF NOT EXISTS 'reel_commented';
ALTER TYPE notification_type ADD VALUE IF NOT EXISTS 'tag';
ALTER TYPE notification_type ADD VALUE IF NOT EXISTS 'share';
ALTER TYPE notification_type ADD VALUE IF NOT EXISTS 'live';
ALTER TYPE notification_type ADD VALUE IF NOT EXISTS 'business';
ALTER TYPE notification_type ADD VALUE IF NOT EXISTS 'system';

-- Add engagement and watch time columns to posts and reels
ALTER TABLE posts ADD COLUMN engagement_score FLOAT NOT NULL DEFAULT 0.0;
ALTER TABLE posts ADD COLUMN watch_time_score FLOAT NOT NULL DEFAULT 0.0;

ALTER TABLE reels ADD COLUMN engagement_score FLOAT NOT NULL DEFAULT 0.0;
ALTER TABLE reels ADD COLUMN watch_time_score FLOAT NOT NULL DEFAULT 0.0;



