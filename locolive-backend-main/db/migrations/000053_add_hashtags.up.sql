-- Hashtags table
CREATE TABLE hashtags (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) UNIQUE NOT NULL,
    usage_count INT NOT NULL DEFAULT 0,
    reels_count INT NOT NULL DEFAULT 0,
    last_used_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_hashtags_name ON hashtags (name);
CREATE INDEX idx_hashtags_usage_count ON hashtags (usage_count DESC);
CREATE INDEX idx_hashtags_last_used ON hashtags (last_used_at DESC);

-- Reel Hashtags Junction Table
CREATE TABLE reel_hashtags (
    reel_id UUID NOT NULL REFERENCES reels(id) ON DELETE CASCADE,
    hashtag_id UUID NOT NULL REFERENCES hashtags(id) ON DELETE CASCADE,
    PRIMARY KEY (reel_id, hashtag_id)
);

CREATE INDEX idx_reel_hashtags_hashtag_id ON reel_hashtags (hashtag_id);
