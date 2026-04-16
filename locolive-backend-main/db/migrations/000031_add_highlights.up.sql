-- Highlight Groups: Named collections of archived stories (saved forever)
CREATE TABLE highlight_groups (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    cover_url TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_highlight_groups_user ON highlight_groups (user_id, created_at DESC);

-- Link archived stories to a highlight group
CREATE TABLE highlight_stories (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    highlight_id UUID NOT NULL REFERENCES highlight_groups(id) ON DELETE CASCADE,
    archived_story_id UUID NOT NULL REFERENCES archived_stories(id) ON DELETE CASCADE,
    added_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(highlight_id, archived_story_id)
);

CREATE INDEX idx_highlight_stories_highlight ON highlight_stories (highlight_id, added_at ASC);
