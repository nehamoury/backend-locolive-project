-- name: CreateDataExportJob :one
INSERT INTO data_export_jobs (
    user_id, status
) VALUES (
    $1, 'pending'
) RETURNING *;

-- name: UpdateDataExportJob :one
UPDATE data_export_jobs
SET status = $2, file_path = $3, expires_at = $4, error_message = $5, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: GetDataExportJob :one
SELECT * FROM data_export_jobs WHERE id = $1;

-- name: GetLatestDataExportJob :one
SELECT * FROM data_export_jobs
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT 1;
