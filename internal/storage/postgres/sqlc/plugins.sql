-- name: GetPluginByID :one
SELECT * FROM plugins
WHERE id = $1;

-- name: ListPlugins :many
SELECT * FROM plugins
ORDER BY updated_at DESC;

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
AND ($2::text IS NULL OR f.plugin_id = $2)
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