-- name: CreateSupportTicket :one
INSERT INTO support_tickets (
    user_id, subject, description, priority
) VALUES (
    $1, $2, $3, $4
) RETURNING *;

-- name: GetUserSupportTickets :many
SELECT * FROM support_tickets
WHERE user_id = $1
ORDER BY created_at DESC;

-- name: UpdateSupportTicketStatus :one
UPDATE support_tickets
SET status = $2, updated_at = NOW()
WHERE id = $1
RETURNING *;
