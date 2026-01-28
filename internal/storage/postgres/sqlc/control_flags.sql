-- Control Flags table queries (kill switch functionality)

-- name: GetControlFlag :one
SELECT * FROM control_flags
WHERE key = $1;

-- name: GetControlFlagsByKeys :many
SELECT * FROM control_flags
WHERE key = ANY($1::text[]);

-- name: UpsertControlFlag :exec
INSERT INTO control_flags (key, enabled, updated_at)
VALUES ($1, $2, NOW())
ON CONFLICT (key) DO UPDATE
SET enabled = EXCLUDED.enabled, updated_at = NOW();

-- name: UpsertControlFlagReturning :one
INSERT INTO control_flags (key, enabled, updated_at)
VALUES ($1, $2, NOW())
ON CONFLICT (key) DO UPDATE
SET enabled = EXCLUDED.enabled, updated_at = NOW()
RETURNING *;
