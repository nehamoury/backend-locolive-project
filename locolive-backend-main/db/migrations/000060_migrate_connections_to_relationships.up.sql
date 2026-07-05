-- Migrate existing followers (accepted and pending)
INSERT INTO relationships (user_id, target_user_id, type, status, created_at)
SELECT 
    requester_id, 
    target_id, 
    'follow'::relationship_type, 
    CASE 
        WHEN status = 'accepted' THEN 'active'::relationship_status
        WHEN status = 'pending' THEN 'pending'::relationship_status
        ELSE 'active'::relationship_status
    END,
    created_at
FROM connections
WHERE status IN ('accepted', 'pending')
ON CONFLICT (user_id, target_user_id, type) DO NOTHING;

-- Migrate existing blocks
INSERT INTO relationships (user_id, target_user_id, type, status, created_at)
SELECT 
    requester_id, 
    target_id, 
    'block'::relationship_type, 
    'active'::relationship_status,
    created_at
FROM connections
WHERE status = 'blocked'
ON CONFLICT (user_id, target_user_id, type) DO NOTHING;
