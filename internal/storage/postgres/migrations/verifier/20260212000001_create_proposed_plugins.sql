-- +goose Up
-- +goose StatementBegin

CREATE TYPE proposed_plugin_status AS ENUM ('submitted', 'approved', 'paid');

CREATE TABLE proposed_plugins (
    public_key TEXT NOT NULL,
    plugin_id TEXT NOT NULL,
    title VARCHAR(255) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    server_endpoint TEXT NOT NULL DEFAULT '',
    category plugin_category NOT NULL DEFAULT 'app',
    status proposed_plugin_status NOT NULL DEFAULT 'submitted',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (public_key, plugin_id)
);

ALTER TABLE plugins DROP COLUMN status;
DROP TYPE plugin_status;

-- +goose StatementEnd

-- +goose Down
