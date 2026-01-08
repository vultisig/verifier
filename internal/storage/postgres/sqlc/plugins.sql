-- name: GetPluginByID :one
SELECT * FROM plugins
WHERE id = $1;