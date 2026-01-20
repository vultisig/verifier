-- +goose Up
CREATE TYPE plugin_owner_added_via AS ENUM ('bootstrap_plugin_key', 'owner_api', 'admin_cli');
CREATE TYPE plugin_owner_role AS ENUM ('admin');

CREATE TABLE plugin_owners (
    plugin_id plugin_id NOT NULL REFERENCES plugins(id) ON DELETE CASCADE,
    public_key TEXT NOT NULL,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    role plugin_owner_role NOT NULL DEFAULT 'admin',
    added_via plugin_owner_added_via NOT NULL,
    added_by_public_key TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (plugin_id, public_key)
);

CREATE INDEX idx_plugin_owners_public_key ON plugin_owners(public_key);

-- +goose Down
DROP TABLE IF EXISTS plugin_owners;
DROP TYPE IF EXISTS plugin_owner_role;
DROP TYPE IF EXISTS plugin_owner_added_via;
