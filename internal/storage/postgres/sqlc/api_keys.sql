-- Plugin API Keys table queries

-- name: GetPluginApiKeys :many
SELECT * FROM plugin_apikey
WHERE plugin_id = $1
ORDER BY created_at DESC;

-- name: GetPluginApiKeyByID :one
SELECT * FROM plugin_apikey
WHERE id = $1;

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

-- name: CountActiveApiKeys :one
SELECT COUNT(*) FROM plugin_apikey
WHERE plugin_id = $1
  AND status = 1
  AND (expires_at IS NULL OR expires_at > NOW());

-- name: CreatePluginApiKeyWithLimit :one
INSERT INTO plugin_apikey (plugin_id, apikey, expires_at, status)
SELECT $1, $2, $3, 1
WHERE (SELECT COUNT(*) FROM plugin_apikey
       WHERE plugin_id = $1 AND status = 1 AND (expires_at IS NULL OR expires_at > NOW())) < sqlc.arg(max_keys)::int
RETURNING *;
