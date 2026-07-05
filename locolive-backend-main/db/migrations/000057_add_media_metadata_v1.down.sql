ALTER TABLE posts
    DROP COLUMN IF EXISTS width,
    DROP COLUMN IF EXISTS height,
    DROP COLUMN IF EXISTS aspect_ratio,
    DROP COLUMN IF EXISTS thumbnail_url,
    DROP COLUMN IF EXISTS blur_hash,
    DROP COLUMN IF EXISTS duration,
    DROP COLUMN IF EXISTS file_size,
    DROP COLUMN IF EXISTS mime_type;

ALTER TABLE reels
    DROP COLUMN IF EXISTS width,
    DROP COLUMN IF EXISTS height,
    DROP COLUMN IF EXISTS aspect_ratio,
    DROP COLUMN IF EXISTS thumbnail_url,
    DROP COLUMN IF EXISTS blur_hash,
    DROP COLUMN IF EXISTS duration,
    DROP COLUMN IF EXISTS file_size,
    DROP COLUMN IF EXISTS mime_type;

ALTER TABLE stories
    DROP COLUMN IF EXISTS width,
    DROP COLUMN IF EXISTS height,
    DROP COLUMN IF EXISTS aspect_ratio,
    DROP COLUMN IF EXISTS blur_hash,
    DROP COLUMN IF EXISTS duration,
    DROP COLUMN IF EXISTS file_size,
    DROP COLUMN IF EXISTS mime_type;
