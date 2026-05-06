-- Add verification fields to users table
ALTER TABLE users ADD COLUMN is_email_verified BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE users ADD COLUMN is_phone_verified BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE users ADD COLUMN is_active BOOLEAN NOT NULL DEFAULT false;

-- Table for Email Verification Tokens
CREATE TABLE email_verifications (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    email varchar NOT NULL,
    token varchar NOT NULL UNIQUE,
    expires_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT (now())
);

CREATE INDEX idx_email_verifications_token ON email_verifications(token);
CREATE INDEX idx_email_verifications_user ON email_verifications(user_id);

-- Table for Phone OTP
CREATE TABLE phone_verifications (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    phone varchar NOT NULL,
    code varchar NOT NULL,
    expires_at timestamptz NOT NULL,
    created_at timestamptz NOT NULL DEFAULT (now()),
    attempts int NOT NULL DEFAULT 0
);

CREATE INDEX idx_phone_verifications_phone ON phone_verifications(phone);
CREATE INDEX idx_phone_verifications_user ON phone_verifications(user_id);
