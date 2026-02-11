-- Proposed plugins table queries

-- name: CreateProposedPlugin :one
INSERT INTO proposed_plugins (public_key, plugin_id, title, description, server_endpoint, category)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetProposedPluginsByPublicKey :many
SELECT * FROM proposed_plugins
WHERE public_key = $1
ORDER BY created_at DESC;

-- name: GetProposedPlugin :one
SELECT * FROM proposed_plugins
WHERE public_key = $1 AND plugin_id = $2;

-- name: UpdateProposedPluginStatus :one
UPDATE proposed_plugins
SET status = sqlc.arg(new_status), updated_at = NOW()
WHERE public_key = sqlc.arg(public_key) AND plugin_id = sqlc.arg(plugin_id) AND status = sqlc.arg(current_status)
RETURNING *;
