DROP TABLE IF EXISTS data_export_jobs;
DROP TABLE IF EXISTS notification_settings;
DROP TABLE IF EXISTS user_preferences;
DROP TABLE IF EXISTS support_tickets;

ALTER TABLE users DROP COLUMN IF EXISTS last_password_change;
ALTER TABLE users DROP COLUMN IF EXISTS two_fa_secret;
ALTER TABLE users DROP COLUMN IF EXISTS two_fa_enabled;
