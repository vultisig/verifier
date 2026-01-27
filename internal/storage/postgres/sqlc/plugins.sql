-- name: GetPluginByID :one
SELECT * FROM plugins
WHERE id = $1;

-- name: ListPlugins :many
SELECT * FROM plugins
ORDER BY updated_at DESC;

-- name: ListPluginsByOwner :many
SELECT p.* FROM plugins p
JOIN plugin_owners po ON p.id::text = po.plugin_id::text
WHERE po.public_key = $1 AND po.active = true
ORDER BY p.updated_at DESC;

-- name: GetPluginByIDAndOwner :one
SELECT p.* FROM plugins p
JOIN plugin_owners po ON p.id::text = po.plugin_id::text
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

-- name: GetPluginApiKeys :many
SELECT * FROM plugin_apikey
WHERE plugin_id = $1
ORDER BY created_at DESC;

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

-- name: GetPluginApiKeyByID :one
SELECT * FROM plugin_apikey
WHERE id = $1;

-- name: GetPluginOwner :one
SELECT * FROM plugin_owners
WHERE plugin_id = $1 AND public_key = $2 AND active = true;

-- name: ListPluginOwners :many
SELECT * FROM plugin_owners
WHERE plugin_id = $1 AND active = true
ORDER BY created_at ASC;

-- name: GetEarningsByPluginOwner :many
SELECT
    f.id,
    f.plugin_id,
    p.title as plugin_name,
    f.amount,
    'usdc' as asset,
    COALESCE(ppb.type::text, 'per-tx') as pricing_type,
    f.created_at,
    f.public_key as from_address,
    COALESCE(ti.tx_hash, '') as tx_hash,
    CASE
        WHEN ti.status_onchain = 'SUCCESS' THEN 'completed'
        WHEN ti.status_onchain = 'FAIL' THEN 'failed'
        ELSE 'pending'
    END as status
FROM fees f
JOIN plugins p ON f.plugin_id::plugin_id = p.id
LEFT JOIN plugin_policies pp ON f.policy_id = pp.id
LEFT JOIN plugin_policy_billing ppb ON pp.id = ppb.plugin_policy_id
LEFT JOIN tx_indexer ti ON f.policy_id = ti.policy_id
WHERE f.plugin_id IN (
    SELECT po.plugin_id::text FROM plugin_owners po WHERE po.public_key = $1 AND po.active = true
)
AND f.transaction_type = 'debit'
ORDER BY f.created_at DESC;

-- name: GetEarningsByPluginOwnerFiltered :many
SELECT
    f.id,
    f.plugin_id,
    p.title as plugin_name,
    f.amount,
    'usdc' as asset,
    COALESCE(ppb.type::text, 'per-tx') as pricing_type,
    f.created_at,
    f.public_key as from_address,
    COALESCE(ti.tx_hash, '') as tx_hash,
    CASE
        WHEN ti.status_onchain = 'SUCCESS' THEN 'completed'
        WHEN ti.status_onchain = 'FAIL' THEN 'failed'
        ELSE 'pending'
    END as status
FROM fees f
JOIN plugins p ON f.plugin_id::plugin_id = p.id
LEFT JOIN plugin_policies pp ON f.policy_id = pp.id
LEFT JOIN plugin_policy_billing ppb ON pp.id = ppb.plugin_policy_id
LEFT JOIN tx_indexer ti ON f.policy_id = ti.policy_id
WHERE f.plugin_id IN (
    SELECT po.plugin_id::text FROM plugin_owners po WHERE po.public_key = $1 AND po.active = true
)
AND f.transaction_type = 'debit'
AND (NULLIF($2, '')::text IS NULL OR f.plugin_id = $2)
AND ($3::timestamptz IS NULL OR f.created_at >= $3)
AND ($4::timestamptz IS NULL OR f.created_at <= $4)
ORDER BY f.created_at DESC;

-- name: GetEarningsSummaryByPluginOwner :one
SELECT
    COALESCE(SUM(f.amount), 0)::bigint as total_earnings,
    COUNT(f.id)::bigint as total_transactions
FROM fees f
WHERE f.plugin_id IN (
    SELECT po.plugin_id::text FROM plugin_owners po WHERE po.public_key = $1 AND po.active = true
)
AND f.transaction_type = 'debit';

-- name: GetEarningsByPluginForOwner :many
SELECT
    f.plugin_id,
    COALESCE(SUM(f.amount), 0)::bigint as total
FROM fees f
WHERE f.plugin_id IN (
    SELECT po.plugin_id::text FROM plugin_owners po WHERE po.public_key = $1 AND po.active = true
)
AND f.transaction_type = 'debit'
GROUP BY f.plugin_id;

-- Team Management Queries

-- name: ListPluginTeamMembers :many
-- List team members for a plugin, excluding staff role (admins can't see staff)
SELECT * FROM plugin_owners
WHERE plugin_id = $1 AND active = true AND role != 'staff'
ORDER BY
    CASE role
        WHEN 'admin' THEN 1
        WHEN 'editor' THEN 2
        WHEN 'viewer' THEN 3
    END,
    created_at ASC;

-- name: GetPluginOwnerWithRole :one
-- Get a specific owner with their role for permission checks
SELECT * FROM plugin_owners
WHERE plugin_id = $1 AND public_key = $2 AND active = true;

-- name: CheckLinkIdUsed :one
-- Check if a link_id has already been used
SELECT EXISTS(SELECT 1 FROM plugin_owners WHERE link_id = $1) as used;

-- name: AddPluginTeamMember :one
-- Add a new team member via magic link invite
INSERT INTO plugin_owners (plugin_id, public_key, role, added_via, added_by_public_key, link_id)
VALUES ($1, $2, $3, 'magic_link', $4, $5)
RETURNING *;

-- name: RemovePluginTeamMember :exec
-- Deactivate a team member (soft delete)
UPDATE plugin_owners
SET active = false, updated_at = NOW()
WHERE plugin_id = $1 AND public_key = $2 AND role != 'staff';