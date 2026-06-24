-- Categories table
CREATE TABLE categories (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) UNIQUE NOT NULL,
    icon VARCHAR(255),
    color VARCHAR(50),
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_categories_slug ON categories(slug);
CREATE INDEX idx_categories_is_active ON categories(is_active);

-- Category stats table
CREATE TABLE category_stats (
    category_id UUID PRIMARY KEY REFERENCES categories(id) ON DELETE CASCADE,
    posts_count INT NOT NULL DEFAULT 0,
    reels_count INT NOT NULL DEFAULT 0,
    stories_count INT NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Alter posts to add category_id
ALTER TABLE posts ADD COLUMN category_id UUID REFERENCES categories(id) ON DELETE SET NULL;
CREATE INDEX idx_posts_category_id ON posts(category_id);

-- Alter reels to add category_id
ALTER TABLE reels ADD COLUMN category_id UUID REFERENCES categories(id) ON DELETE SET NULL;
CREATE INDEX idx_reels_category_id ON reels(category_id);

-- Alter hashtags to add slug
ALTER TABLE hashtags ADD COLUMN slug VARCHAR(255) UNIQUE;

-- We need a function to generate slugs or handle existing ones, but since it's nullable we can just set it uniquely going forward.
-- To avoid null issues if they are queried, we can set slug = lower(regexp_replace(name, '[^a-zA-Z0-9]', '', 'g')) where slug is null.
UPDATE hashtags SET slug = lower(regexp_replace(name, '[^a-zA-Z0-9]+', '-', 'g')) WHERE slug IS NULL;

-- Create post_hashtags junction table
CREATE TABLE post_hashtags (
    post_id UUID NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    hashtag_id UUID NOT NULL REFERENCES hashtags(id) ON DELETE CASCADE,
    PRIMARY KEY (post_id, hashtag_id)
);

CREATE INDEX idx_post_hashtags_hashtag_id ON post_hashtags (hashtag_id);
