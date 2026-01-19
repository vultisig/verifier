-- +goose Up
CREATE TABLE plugin_owners (
    plugin_id plugin_id NOT NULL REFERENCES plugins(id) ON DELETE CASCADE,
    public_key TEXT NOT NULL,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    added_via TEXT NOT NULL,
    added_by_public_key TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (plugin_id, public_key)
);

CREATE INDEX idx_plugin_owners_public_key ON plugin_owners(public_key);

-- +goose Down
DROP TABLE IF EXISTS plugin_owners;
