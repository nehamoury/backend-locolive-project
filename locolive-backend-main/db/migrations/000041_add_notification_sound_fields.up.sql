-- Add sub_type and sound to notifications table
ALTER TABLE notifications ADD COLUMN IF NOT EXISTS sub_type VARCHAR(50);
ALTER TABLE notifications ADD COLUMN IF NOT EXISTS sound VARCHAR(100);

-- Update existing notifications with default values based on type if needed
UPDATE notifications SET sub_type = type::text WHERE sub_type IS NULL;
