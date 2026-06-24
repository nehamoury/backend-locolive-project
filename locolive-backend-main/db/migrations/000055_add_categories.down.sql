-- Drop post_hashtags
DROP TABLE IF EXISTS post_hashtags CASCADE;

-- Drop slug from hashtags
ALTER TABLE hashtags DROP COLUMN IF EXISTS slug;

-- Drop category_id from reels
ALTER TABLE reels DROP COLUMN IF EXISTS category_id;

-- Drop category_id from posts
ALTER TABLE posts DROP COLUMN IF EXISTS category_id;

-- Drop category_stats
DROP TABLE IF EXISTS category_stats CASCADE;

-- Drop categories
DROP TABLE IF EXISTS categories CASCADE;
