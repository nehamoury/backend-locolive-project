-- Fixed test data for privacy_social DB (locations table + stories)
-- Create test users if not exist
INSERT INTO users (id, username, full_name, email, password_hash, is_ghost_mode, created_at) VALUES
(gen_random_uuid(), 'nearby_user1', 'Nearby User 1', 'user1@test.com', '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi', false, NOW()),
(gen_random_uuid(), 'nearby_user2', 'Nearby User 2', 'user2@test.com', '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi', false, NOW()),
(gen_random_uuid(), 'story_user', 'Story User', 'story@test.com', '$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi', false, NOW())
ON CONFLICT DO NOTHING;

-- Add test locations (within Delhi)
INSERT INTO locations (user_id, geohash, lng, lat, time_bucket, expires_at) VALUES
((SELECT id FROM users WHERE username='nearby_user1'), 'tu1sw0j', 77.2090, 28.6139, date_trunc('minute', NOW() - INTERVAL '5 minutes'), NOW() + INTERVAL '24 hours'),
((SELECT id FROM users WHERE username='nearby_user2'), 'tu1sw5p', 77.2150, 28.6200, date_trunc('minute', NOW() - INTERVAL '3 minutes'), NOW() + INTERVAL '24 hours'),
((SELECT id FROM users WHERE username='story_user'), 'tu1svqy', 77.2000, 28.6100, date_trunc('minute', NOW() - INTERVAL '2 minutes'), NOW() + INTERVAL '24 hours')
ON CONFLICT (user_id, time_bucket) DO UPDATE SET lng=EXCLUDED.lng, lat=EXCLUDED.lat, expires_at=EXCLUDED.expires_at;

-- Add test stories
INSERT INTO stories (id, user_id, media_url, media_type, latitude, longitude, caption, expires_at, created_at, show_location) VALUES
(gen_random_uuid(), (SELECT id FROM users WHERE username='story_user'), '/test/story1.jpg', 'image', 28.6100, 77.2000, 'Test story nearby Delhi!', NOW() + INTERVAL '24 hours', NOW() - INTERVAL '2 minutes', true),
(gen_random_uuid(), (SELECT id FROM users WHERE username='story_user'), '/test/story2.jpg', 'image', 28.6100, 77.2000, 'Live from Delhi!', NOW() + INTERVAL '24 hours', NOW() - INTERVAL '1 minute', true)
ON CONFLICT DO NOTHING;

