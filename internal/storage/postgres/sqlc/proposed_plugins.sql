-- Proposed plugins table queries

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

-- name: GetProposedPluginByID :one
SELECT * FROM proposed_plugins
WHERE plugin_id = $1;

-- name: MarkProposedPluginListed :execrows
UPDATE proposed_plugins
SET status = 'listed', updated_at = NOW()
WHERE plugin_id = $1 AND status = 'approved';

-- name: ListProposedPluginImages :many
SELECT * FROM proposed_plugin_images
WHERE plugin_id = $1 AND deleted = false AND visible = true
ORDER BY image_type, image_order ASC;

-- name: InsertPluginImage :exec
INSERT INTO plugin_images (id, plugin_id, image_type, s3_path, image_order, uploaded_by_public_key, visible, deleted, content_type, filename, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW());
