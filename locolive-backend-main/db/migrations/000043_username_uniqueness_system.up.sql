-- Username Uniqueness System Migration
-- Ensures case-insensitive uniqueness and adds history tracking

-- Add normalized username column for storing original input
ALTER TABLE users ADD COLUMN IF NOT EXISTS username_normalized varchar(20);

-- Update existing usernames to lowercase (idempotent operation)
UPDATE users SET username_normalized = LOWER(username) WHERE username_normalized IS NULL;

-- Create case-insensitive unique index for fast lookups
-- This ensures 'Rahul', 'rahul', 'RAHUL' are all treated as the same
CREATE UNIQUE INDEX IF NOT EXISTS idx_username_lower ON users (LOWER(username));

-- Create index for username search performance
CREATE INDEX IF NOT EXISTS idx_username_trgm ON users USING gin (username gin_trgm_ops);

-- Username history table for tracking changes
CREATE TABLE IF NOT EXISTS username_history (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    old_username varchar(20) NOT NULL,
    new_username varchar(20) NOT NULL,
    changed_at timestamptz NOT NULL DEFAULT (now()),
    changed_by uuid REFERENCES users(id) ON DELETE SET NULL
);

-- Index for user lookups
CREATE INDEX IF NOT EXISTS idx_username_history_user_id ON username_history(user_id);
CREATE INDEX IF NOT EXISTS idx_username_history_old_username ON username_history(old_username);

-- Reserved usernames table for admin-controlled blocked names
CREATE TABLE IF NOT EXISTS reserved_usernames (
    username varchar(20) PRIMARY KEY,
    reason varchar(100) NOT NULL DEFAULT 'reserved',
    created_at timestamptz NOT NULL DEFAULT (now())
);

-- Insert default reserved usernames
INSERT INTO reserved_usernames (username, reason) VALUES
    ('admin', 'system_reserved'),
    ('administrator', 'system_reserved'),
    ('support', 'system_reserved'),
    ('help', 'system_reserved'),
    ('official', 'system_reserved'),
    ('locolive', 'trademark'),
    ('system', 'system_reserved'),
    ('root', 'system_reserved'),
    ('superuser', 'system_reserved'),
    ('moderator', 'system_reserved'),
    ('mod', 'system_reserved'),
    ('staff', 'system_reserved'),
    ('team', 'system_reserved'),
    ('security', 'system_reserved'),
    ('billing', 'system_reserved'),
    ('payment', 'system_reserved'),
    ('api', 'system_reserved'),
    ('dev', 'system_reserved'),
    ('developer', 'system_reserved'),
    ('test', 'system_reserved'),
    ('testing', 'system_reserved'),
    ('null', 'system_reserved'),
    ('undefined', 'system_reserved'),
    ('unknown', 'system_reserved'),
    ('guest', 'system_reserved'),
    ('anonymous', 'system_reserved'),
    ('user', 'system_reserved'),
    ('users', 'system_reserved'),
    ('account', 'system_reserved'),
    ('accounts', 'system_reserved'),
    ('login', 'system_reserved'),
    ('logout', 'system_reserved'),
    ('signup', 'system_reserved'),
    ('register', 'system_reserved'),
    ('password', 'system_reserved'),
    ('reset', 'system_reserved'),
    ('verify', 'system_reserved'),
    ('confirm', 'system_reserved'),
    ('email', 'system_reserved'),
    ('phone', 'system_reserved'),
    ('contact', 'system_reserved'),
    ('about', 'system_reserved'),
    ('help', 'system_reserved'),
    ('faq', 'system_reserved'),
    ('terms', 'system_reserved'),
    ('privacy', 'system_reserved'),
    ('policy', 'system_reserved'),
    ('legal', 'system_reserved'),
    ('cookies', 'system_reserved'),
    ('settings', 'system_reserved'),
    ('config', 'system_reserved'),
    ('dashboard', 'system_reserved'),
    ('home', 'system_reserved'),
    ('index', 'system_reserved'),
    ('main', 'system_reserved'),
    ('start', 'system_reserved'),
    ('getstarted', 'system_reserved'),
    ('welcome', 'system_reserved'),
    ('hello', 'system_reserved'),
    ('hi', 'system_reserved'),
    ('news', 'system_reserved'),
    ('blog', 'system_reserved'),
    ('careers', 'system_reserved'),
    ('jobs', 'system_reserved'),
    ('press', 'system_reserved'),
    ('media', 'system_reserved'),
    ('partner', 'system_reserved'),
    ('partners', 'system_reserved'),
    ('affiliate', 'system_reserved'),
    ('business', 'system_reserved'),
    ('enterprise', 'system_reserved'),
    ('pro', 'system_reserved'),
    ('premium', 'system_reserved'),
    ('gold', 'system_reserved'),
    ('vip', 'system_reserved'),
    ('elite', 'system_reserved'),
    ('featured', 'system_reserved'),
    ('popular', 'system_reserved'),
    ('trending', 'system_reserved'),
    ('explore', 'system_reserved'),
    ('discover', 'system_reserved'),
    ('search', 'system_reserved'),
    ('find', 'system_reserved'),
    ('lookup', 'system_reserved'),
    ('all', 'system_reserved'),
    ('everyone', 'system_reserved'),
    ('notifications', 'system_reserved'),
    ('messages', 'system_reserved'),
    ('chat', 'system_reserved'),
    ('calls', 'system_reserved'),
    ('stories', 'system_reserved'),
    ('reels', 'system_reserved'),
    ('posts', 'system_reserved'),
    ('photos', 'system_reserved'),
    ('videos', 'system_reserved'),
    ('files', 'system_reserved'),
    ('uploads', 'system_reserved'),
    ('downloads', 'system_reserved'),
    ('share', 'system_reserved'),
    ('send', 'system_reserved'),
    ('create', 'system_reserved'),
    ('edit', 'system_reserved'),
    ('delete', 'system_reserved'),
    ('remove', 'system_reserved'),
    ('block', 'system_reserved'),
    ('report', 'system_reserved'),
    ('feedback', 'system_reserved'),
    ('status', 'system_reserved'),
    ('online', 'system_reserved'),
    ('offline', 'system_reserved'),
    ('away', 'system_reserved'),
    ('busy', 'system_reserved')
ON CONFLICT (username) DO NOTHING;
