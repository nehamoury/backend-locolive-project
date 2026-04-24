-- Add new notification types
ALTER TYPE notification_type ADD VALUE IF NOT EXISTS 'badge';
ALTER TYPE notification_type ADD VALUE IF NOT EXISTS 'streak';
ALTER TYPE notification_type ADD VALUE IF NOT EXISTS 'nudge';
ALTER TYPE notification_type ADD VALUE IF NOT EXISTS 'nearby_story';
ALTER TYPE notification_type ADD VALUE IF NOT EXISTS 'reel_liked';
ALTER TYPE notification_type ADD VALUE IF NOT EXISTS 'reel_commented';

-- Notification preferences table
CREATE TABLE IF NOT EXISTS notification_preferences (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  push_enabled BOOLEAN DEFAULT true,
  email_enabled BOOLEAN DEFAULT true,
  crossing_alerts BOOLEAN DEFAULT true,
  message_alerts BOOLEAN DEFAULT true,
  story_alerts BOOLEAN DEFAULT true,
  updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- User streaks table
CREATE TABLE IF NOT EXISTS user_streaks (
  user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  current_streak INT NOT NULL DEFAULT 0,
  longest_streak INT NOT NULL DEFAULT 0,
  last_activity_date DATE,
  updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Badges table
CREATE TABLE IF NOT EXISTS badges (
  id VARCHAR(50) PRIMARY KEY,
  name VARCHAR(100) NOT NULL,
  description TEXT NOT NULL,
  icon VARCHAR(50) NOT NULL,
  category VARCHAR(50) NOT NULL,
  requirement JSONB NOT NULL,
  created_at TIMESTAMPTZ DEFAULT NOW()
);

-- User earned badges
CREATE TABLE IF NOT EXISTS user_badges (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  badge_id VARCHAR(50) NOT NULL REFERENCES badges(id) ON DELETE CASCADE,
  earned_at TIMESTAMPTZ DEFAULT NOW(),
  UNIQUE(user_id, badge_id)
);

-- Daily engagement stats
CREATE TABLE IF NOT EXISTS daily_stats (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  date DATE NOT NULL,
  crossings_count INT DEFAULT 0,
  stories_posted INT DEFAULT 0,
  messages_sent INT DEFAULT 0,
  locations_updated INT DEFAULT 0,
  created_at TIMESTAMPTZ DEFAULT NOW(),
  UNIQUE(user_id, date)
);

-- Engagement events for analytics/badges
CREATE TABLE IF NOT EXISTS engagement_events (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  event_type VARCHAR(50) NOT NULL,
  event_data JSONB,
  created_at TIMESTAMPTZ DEFAULT NOW()
);
