-- +goose Up
-- +goose StatementBegin

CREATE TYPE plugin_status AS ENUM (
  'draft',
  'staging_approved',
  'submitted',
  'approved',
  'listed'
);

ALTER TABLE plugins
  ADD COLUMN status plugin_status NOT NULL DEFAULT 'listed';

ALTER TABLE plugins
  ALTER COLUMN status SET DEFAULT 'draft';

ALTER TYPE plugin_owner_added_via
  ADD VALUE IF NOT EXISTS 'portal_create';

CREATE TYPE portal_approver_role AS ENUM ('staging_approver', 'listing_approver', 'admin');
CREATE TYPE portal_approver_added_via AS ENUM ('bootstrap', 'admin_portal', 'cli');

CREATE TABLE portal_approvers (
  public_key TEXT PRIMARY KEY,
  role portal_approver_role NOT NULL DEFAULT 'staging_approver',
  active BOOLEAN NOT NULL DEFAULT TRUE,
  added_via portal_approver_added_via NOT NULL DEFAULT 'bootstrap',
  added_by_public_key TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose StatementEnd

-- +goose Down
