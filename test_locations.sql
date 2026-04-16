-- Fix recent locations to Delhi area for testing
UPDATE locations 
SET 
  geohash = 'tu1sw0j',
  geom = ST_Point(77.2090, 28.6139),
  expires_at = NOW() + INTERVAL '1 day'
WHERE id IN (SELECT id FROM locations ORDER BY created_at DESC LIMIT 3);

-- Add test story for recent user
INSERT INTO stories (user_id, media_url, media_type, geohash, geom, caption, visibility, expires_at) VALUES
((SELECT user_id FROM locations ORDER BY created_at DESC LIMIT 1), 
 '/test.jpg', 'image', 'tu1sw0j', ST_Point(77.2090, 28.6139), 'Test story for map!', 'public', NOW() + INTERVAL '1 day');

