-- Reels: Full-screen vertical video content
CREATE TABLE reels (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    video_url TEXT NOT NULL,
    caption TEXT,
    is_ai_generated BOOLEAN NOT NULL DEFAULT false,
    location_name TEXT,
    geohash TEXT,
    geom GEOMETRY(Point, 4326),
    likes_count INT NOT NULL DEFAULT 0,
    comments_count INT NOT NULL DEFAULT 0,
    shares_count INT NOT NULL DEFAULT 0,
    saves_count INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_reels_user_id ON reels (user_id, created_at DESC);
CREATE INDEX idx_reels_geom ON reels USING GIST (geom) WHERE geom IS NOT NULL;
CREATE INDEX idx_reels_created_at ON reels (created_at DESC);
CREATE INDEX idx_reels_ai_generated ON reels (is_ai_generated) WHERE is_ai_generated = true;

-- Reel Likes
CREATE TABLE reel_likes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    reel_id UUID NOT NULL REFERENCES reels(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(reel_id, user_id)
);

CREATE INDEX idx_reel_likes_reel ON reel_likes (reel_id);
CREATE INDEX idx_reel_likes_user ON reel_likes (user_id);

-- Reel Comments
CREATE TABLE reel_comments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    reel_id UUID NOT NULL REFERENCES reels(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_reel_comments_reel ON reel_comments (reel_id, created_at ASC);

-- Reel Saves
CREATE TABLE reel_saves (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    reel_id UUID NOT NULL REFERENCES reels(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(reel_id, user_id)
);

CREATE INDEX idx_reel_saves_user ON reel_saves (user_id, created_at DESC);
