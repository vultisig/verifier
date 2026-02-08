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

-- name: GetPluginPricings :many
SELECT * FROM pricings
WHERE plugin_id = $1
ORDER BY created_at DESC;
