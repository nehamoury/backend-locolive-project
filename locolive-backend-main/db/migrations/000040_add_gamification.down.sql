DROP TABLE IF EXISTS engagement_events;
DROP TABLE IF EXISTS daily_stats;
DROP TABLE IF EXISTS user_badges;
DROP TABLE IF EXISTS badges;
DROP TABLE IF EXISTS user_streaks;
DROP TABLE IF EXISTS notification_preferences;

-- Note: Removing values from an ENUM is not directly supported in Postgres easily without recreating the type.
-- Usually, we leave it or recreate the type if absolutely necessary.
