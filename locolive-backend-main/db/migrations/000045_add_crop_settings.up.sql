-- Add crop_settings column to posts and stories
ALTER TABLE posts ADD COLUMN IF NOT EXISTS crop_settings JSONB;
ALTER TABLE stories ADD COLUMN IF NOT EXISTS crop_settings JSONB;
