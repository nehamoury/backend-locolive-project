ALTER TABLE notifications ADD COLUMN related_post_id UUID REFERENCES posts(id) ON DELETE CASCADE;
ALTER TABLE notifications ADD COLUMN related_reel_id UUID REFERENCES reels(id) ON DELETE CASCADE;
