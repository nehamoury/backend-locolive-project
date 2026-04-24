-- name: GetUserStreak :one
SELECT * FROM user_streaks
WHERE user_id = $1 LIMIT 1;

-- name: UpdateUserStreak :one
INSERT INTO user_streaks (
  user_id,
  current_streak,
  longest_streak,
  last_activity_date,
  updated_at
) VALUES (
  $1, $2, $3, $4, NOW()
) ON CONFLICT (user_id) DO UPDATE SET
  current_streak = EXCLUDED.current_streak,
  longest_streak = EXCLUDED.longest_streak,
  last_activity_date = EXCLUDED.last_activity_date,
  updated_at = NOW()
RETURNING *;

-- name: IncrementDailyStats :one
INSERT INTO daily_stats (
  user_id,
  date,
  crossings_count,
  stories_posted,
  messages_sent,
  locations_updated
) VALUES (
  $1, $2, 
  COALESCE(sqlc.narg('crossings_count'), 0), 
  COALESCE(sqlc.narg('stories_posted'), 0), 
  COALESCE(sqlc.narg('messages_sent'), 0), 
  COALESCE(sqlc.narg('locations_updated'), 0)
) ON CONFLICT (user_id, date) DO UPDATE SET
  crossings_count = daily_stats.crossings_count + COALESCE(EXCLUDED.crossings_count, 0),
  stories_posted = daily_stats.stories_posted + COALESCE(EXCLUDED.stories_posted, 0),
  messages_sent = daily_stats.messages_sent + COALESCE(EXCLUDED.messages_sent, 0),
  locations_updated = daily_stats.locations_updated + COALESCE(EXCLUDED.locations_updated, 0),
  created_at = NOW()
RETURNING *;

-- name: GetDailyStats :many
SELECT * FROM daily_stats
WHERE user_id = $1 AND date >= $2
ORDER BY date DESC;

-- name: CreateBadge :one
INSERT INTO badges (
  id, name, description, icon, category, requirement
) VALUES (
  $1, $2, $3, $4, $5, $6
) RETURNING *;

-- name: GetBadge :one
SELECT * FROM badges
WHERE id = $1 LIMIT 1;

-- name: ListAllBadges :many
SELECT * FROM badges
ORDER BY category, name;

-- name: AwardBadge :one
INSERT INTO user_badges (
  user_id, badge_id, earned_at
) VALUES (
  $1, $2, NOW()
) ON CONFLICT (user_id, badge_id) DO NOTHING
RETURNING *;

-- name: GetUserBadges :many
SELECT b.*, ub.earned_at
FROM badges b
JOIN user_badges ub ON b.id = ub.badge_id
WHERE ub.user_id = $1
ORDER BY ub.earned_at DESC;

-- name: CreateEngagementEvent :one
INSERT INTO engagement_events (
  user_id, event_type, event_data
) VALUES (
  $1, $2, $3
) RETURNING *;

-- name: GetNotificationPreferences :one
SELECT * FROM notification_preferences
WHERE user_id = $1 LIMIT 1;

-- name: UpdateNotificationPreferences :one
INSERT INTO notification_preferences (
  user_id, push_enabled, email_enabled, crossing_alerts, message_alerts, story_alerts
) VALUES (
  $1, $2, $3, $4, $5, $6
) ON CONFLICT (user_id) DO UPDATE SET
  push_enabled = EXCLUDED.push_enabled,
  email_enabled = EXCLUDED.email_enabled,
  crossing_alerts = EXCLUDED.crossing_alerts,
  message_alerts = EXCLUDED.message_alerts,
  story_alerts = EXCLUDED.story_alerts,
  updated_at = NOW()
RETURNING *;
