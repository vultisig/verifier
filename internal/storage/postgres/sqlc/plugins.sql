-- Plugins table queries

-- name: GetPluginByID :one
SELECT * FROM plugins
WHERE id = $1;

-- name: ListPlugins :many
SELECT * FROM plugins
ORDER BY updated_at DESC;

-- name: ListPluginsByOwner :many
SELECT p.* FROM plugins p
JOIN plugin_owners po ON p.id = po.plugin_id
WHERE po.public_key = $1 AND po.active = true
ORDER BY p.updated_at DESC;

-- name: GetPluginByIDAndOwner :one
SELECT p.* FROM plugins p
JOIN plugin_owners po ON p.id = po.plugin_id
WHERE p.id = $1 AND po.public_key = $2 AND po.active = true;

-- name: UpdatePlugin :one
UPDATE plugins
SET
    title = $2,
    description = $3,
    server_endpoint = $4,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: CreateDraftPlugin :one
INSERT INTO plugins (id, title, description, server_endpoint, category, status)
VALUES ($1, $2, '', $3, $4, 'draft')
RETURNING *;

-- name: UpdatePluginStatus :one
UPDATE plugins SET status = sqlc.arg(new_status), updated_at = NOW()
WHERE id = sqlc.arg(id) AND status = sqlc.arg(current_status)
RETURNING *;

-- name: GetPluginPricings :many
SELECT pr.* FROM pricings pr
JOIN plugins p ON pr.plugin_id = p.id
WHERE pr.plugin_id = $1 AND p.status = 'listed'
ORDER BY pr.created_at DESC;
