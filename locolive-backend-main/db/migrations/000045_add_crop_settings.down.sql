-- Remove crop_settings column from posts and stories
ALTER TABLE posts DROP COLUMN IF EXISTS crop_settings;
ALTER TABLE stories DROP COLUMN IF EXISTS crop_settings;
