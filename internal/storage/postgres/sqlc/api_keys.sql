-- Plugin API Keys table queries

-- name: GetPluginApiKeys :many
SELECT * FROM plugin_apikey
WHERE plugin_id = $1
ORDER BY created_at DESC;

-- name: GetPluginApiKeyByID :one
SELECT * FROM plugin_apikey
WHERE id = $1;

-- name: CreatePluginApiKey :one
INSERT INTO plugin_apikey (plugin_id, apikey, expires_at, status)
VALUES ($1, $2, $3, 1)
RETURNING *;

-- name: UpdatePluginApiKeyStatus :one
UPDATE plugin_apikey
SET status = $2
WHERE id = $1
RETURNING *;

-- name: ExpirePluginApiKey :one
UPDATE plugin_apikey
SET expires_at = NOW()
WHERE id = $1
RETURNING *;
