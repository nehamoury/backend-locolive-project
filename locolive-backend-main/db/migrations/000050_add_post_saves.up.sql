-- Add saves_count to posts
ALTER TABLE posts ADD COLUMN saves_count INT NOT NULL DEFAULT 0;

-- Create post_saves table
CREATE TABLE post_saves (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    post_id UUID NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(post_id, user_id)
);

CREATE INDEX idx_post_saves_user ON post_saves (user_id, created_at DESC);
