CREATE TABLE user_fcm_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token TEXT NOT NULL,
    device_type TEXT NOT NULL DEFAULT 'web',
    last_used_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(user_id, token)
);

CREATE INDEX idx_user_fcm_tokens_user_id ON user_fcm_tokens(user_id);
