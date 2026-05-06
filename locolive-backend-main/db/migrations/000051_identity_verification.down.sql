DROP TABLE IF EXISTS phone_verifications;
DROP TABLE IF EXISTS email_verifications;
ALTER TABLE users DROP COLUMN IF EXISTS is_active;
ALTER TABLE users DROP COLUMN IF EXISTS is_phone_verified;
ALTER TABLE users DROP COLUMN IF EXISTS is_email_verified;
