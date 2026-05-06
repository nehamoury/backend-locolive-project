-- name: CreateEmailVerification :one
INSERT INTO email_verifications (
    user_id,
    email,
    token,
    expires_at
) VALUES (
    $1, $2, $3, $4
) RETURNING *;

-- name: GetEmailVerification :one
SELECT * FROM email_verifications
WHERE token = $1 LIMIT 1;

-- name: DeleteEmailVerification :exec
DELETE FROM email_verifications
WHERE user_id = $1;

-- name: CreatePhoneVerification :one
INSERT INTO phone_verifications (
    user_id,
    phone,
    code,
    expires_at
) VALUES (
    $1, $2, $3, $4
) RETURNING *;

-- name: GetLatestPhoneVerification :one
SELECT * FROM phone_verifications
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT 1;

-- name: DeletePhoneVerification :exec
DELETE FROM phone_verifications
WHERE user_id = $1;

-- name: UpdatePhoneVerificationAttempts :one
UPDATE phone_verifications
SET attempts = attempts + 1
WHERE id = $1
RETURNING *;

-- name: VerifyEmail :one
UPDATE users
SET is_email_verified = true,
    is_active = CASE WHEN is_phone_verified = true THEN true ELSE is_active END
WHERE id = $1
RETURNING *;

-- name: VerifyPhone :one
UPDATE users
SET is_phone_verified = true,
    is_active = CASE WHEN is_email_verified = true THEN true ELSE is_active END
WHERE id = $1
RETURNING *;
