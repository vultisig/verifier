-- Plugin Owners table queries (team management)

-- name: GetPluginOwner :one
SELECT * FROM plugin_owners
WHERE plugin_id = $1 AND public_key = $2 AND active = true;

-- name: ListPluginOwners :many
SELECT * FROM plugin_owners
WHERE plugin_id = $1 AND active = true
ORDER BY created_at ASC;

-- Team Management Queries
-- List team members for a plugin, excluding staff role (admins can't see staff)
-- name: ListPluginTeamMembers :many
SELECT * FROM plugin_owners
WHERE plugin_id = $1 AND active = true AND role != 'staff'
ORDER BY
    CASE role
        WHEN 'admin' THEN 1
        WHEN 'editor' THEN 2
        WHEN 'viewer' THEN 3
    END,
    created_at ASC;

-- Get a specific owner with their role for permission checks
-- name: GetPluginOwnerWithRole :one
SELECT * FROM plugin_owners
WHERE plugin_id = $1 AND public_key = $2 AND active = true;

-- Check if a link_id has already been used
-- name: CheckLinkIdUsed :one
SELECT EXISTS(SELECT 1 FROM plugin_owners WHERE link_id = $1) as used;

-- name: AddPluginTeamMember :one
INSERT INTO plugin_owners (plugin_id, public_key, role, added_via, added_by_public_key, link_id)
VALUES ($1, $2, $3, 'magic_link', $4, $5)
ON CONFLICT (plugin_id, public_key) DO UPDATE SET
    active = true,
    role = EXCLUDED.role,
    added_via = EXCLUDED.added_via,
    added_by_public_key = EXCLUDED.added_by_public_key,
    link_id = EXCLUDED.link_id,
    updated_at = NOW()
RETURNING *;

-- Deactivate a team member (soft delete)
-- name: RemovePluginTeamMember :exec
UPDATE plugin_owners
SET active = false, updated_at = NOW()
WHERE plugin_id = $1 AND public_key = $2 AND role != 'staff';

-- name: CreatePluginOwnerFromPortal :one
INSERT INTO plugin_owners (plugin_id, public_key, role, added_via)
VALUES ($1, $2, 'admin', 'portal_create')
ON CONFLICT (plugin_id, public_key) DO UPDATE SET
    active = true,
    role = 'admin',
    added_via = 'portal_create',
    updated_at = NOW()
RETURNING *;
