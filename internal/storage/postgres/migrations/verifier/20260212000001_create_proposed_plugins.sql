-- +goose Up
-- +goose StatementBegin

ALTER TYPE plugin_owner_added_via ADD VALUE IF NOT EXISTS 'portal_create';

CREATE TYPE portal_approver_added_via AS ENUM ('bootstrap', 'admin_portal', 'cli');

CREATE TABLE portal_approvers (
  public_key TEXT PRIMARY KEY,
  active BOOLEAN NOT NULL DEFAULT TRUE,
  added_via portal_approver_added_via NOT NULL DEFAULT 'bootstrap',
  added_by_public_key TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TYPE proposed_plugin_status AS ENUM ('submitted', 'approved', 'listed', 'archived');
CREATE TYPE proposed_plugin_pricing AS ENUM ('free', 'per-tx', 'per-install');

CREATE TABLE proposed_plugins (
    plugin_id TEXT PRIMARY KEY,
    public_key TEXT NOT NULL,
    title VARCHAR(255) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    server_endpoint TEXT NOT NULL,
    category plugin_category NOT NULL DEFAULT 'app',
    supported_chains TEXT[] NOT NULL DEFAULT '{}',
    pricing_model proposed_plugin_pricing,
    contact_email TEXT NOT NULL,
    notes TEXT NOT NULL DEFAULT '',
    status proposed_plugin_status NOT NULL DEFAULT 'submitted',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_proposed_plugins_public_key ON proposed_plugins(public_key);
CREATE INDEX idx_proposed_plugins_status ON proposed_plugins(status);
CREATE INDEX idx_proposed_plugins_public_key_status ON proposed_plugins(public_key, status);

CREATE TABLE proposed_plugin_images (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    plugin_id TEXT NOT NULL REFERENCES proposed_plugins(plugin_id) ON DELETE CASCADE,
    image_type TEXT NOT NULL,
    s3_path TEXT NOT NULL,
    image_order INT NOT NULL DEFAULT 0,
    uploaded_by_public_key TEXT NOT NULL,
    visible BOOLEAN NOT NULL DEFAULT TRUE,
    deleted BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    content_type TEXT NOT NULL,
    filename TEXT NOT NULL DEFAULT '',

    CONSTRAINT proposed_plugin_images_content_type_check
      CHECK (content_type = ANY (ARRAY['image/jpeg','image/png','image/webp'])),

    CONSTRAINT proposed_plugin_images_image_type_check
      CHECK (image_type = ANY (ARRAY['logo','banner','thumbnail','media']))
);

CREATE INDEX proposed_plugin_images_plugin_id_idx ON proposed_plugin_images(plugin_id);
CREATE INDEX proposed_plugin_images_plugin_id_type_idx ON proposed_plugin_images(plugin_id, image_type);

-- One logo, one banner, one thumbnail per proposal
CREATE UNIQUE INDEX proposed_plugin_images_one_logo
  ON proposed_plugin_images(plugin_id)
  WHERE image_type = 'logo' AND deleted = FALSE;

CREATE UNIQUE INDEX proposed_plugin_images_one_banner
  ON proposed_plugin_images(plugin_id)
  WHERE image_type = 'banner' AND deleted = FALSE;

CREATE UNIQUE INDEX proposed_plugin_images_one_thumbnail
  ON proposed_plugin_images(plugin_id)
  WHERE image_type = 'thumbnail' AND deleted = FALSE;

-- +goose StatementEnd

-- +goose Down
