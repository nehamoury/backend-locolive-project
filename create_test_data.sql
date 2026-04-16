-- Test data for nearby users & stories (run: psql -d locolive_db -f create_test_data.sql)
INSERT INTO users (id, username, latitude, longitude, is_active, last_seen, created_at) VALUES
('test1', 'nearby_user1', 28.6139, 77.2090, true, NOW(), NOW()),
('test2', 'nearby_user2', 28.6200, 77.2150, true, NOW(), NOW()),
('test3', 'story_user', 28.6100, 77.2000, true, NOW(), NOW())
ON CONFLICT (id) DO UPDATE SET latitude=EXCLUDED.latitude, longitude=EXCLUDED.longitude, is_active=true, last_seen=NOW();

INSERT INTO stories (id, user_id, media_url, media_type, latitude, longitude, caption, expires_at, created_at, show_location) VALUES
('story1', 'test3', '/test/story1.jpg', 'image', 28.6100, 77.2000, 'Test story nearby!', NOW() + INTERVAL '24 hours', NOW(), true),
('story2', 'test3', '/test/story2.jpg', 'image', 28.6100, 77.2000, 'Another story!', NOW() + INTERVAL '24 hours', NOW(), true)
ON CONFLICT (id) DO NOTHING;

