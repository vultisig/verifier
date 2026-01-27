-- +goose Up
-- Add new roles to the plugin_owner_role enum
ALTER TYPE plugin_owner_role ADD VALUE IF NOT EXISTS 'staff';
ALTER TYPE plugin_owner_role ADD VALUE IF NOT EXISTS 'editor';
ALTER TYPE plugin_owner_role ADD VALUE IF NOT EXISTS 'viewer';

-- Add magic link invite via type
ALTER TYPE plugin_owner_added_via ADD VALUE IF NOT EXISTS 'magic_link';

-- Add link_id column to track magic link usage (ensures one-time use)
ALTER TABLE plugin_owners ADD COLUMN IF NOT EXISTS link_id UUID;

-- Create unique index on link_id (allows NULLs but enforces uniqueness for non-NULL values)
CREATE UNIQUE INDEX IF NOT EXISTS idx_plugin_owners_link_id ON plugin_owners(link_id) WHERE link_id IS NOT NULL;

-- +goose Down
-- Note: PostgreSQL doesn't support removing enum values, so we only drop the column and index
DROP INDEX IF EXISTS idx_plugin_owners_link_id;
ALTER TABLE plugin_owners DROP COLUMN IF EXISTS link_id;
