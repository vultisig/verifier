-- +goose Up
CREATE TABLE IF NOT EXISTS plugin_installations (
    plugin_id     plugin_id        NOT NULL,
    public_key    TEXT             NOT NULL,
    installed_at  TIMESTAMPTZ      NOT NULL DEFAULT NOW(),
    PRIMARY KEY (plugin_id, public_key)
);

-- +goose Down
DROP TABLE IF EXISTS plugin_installations;