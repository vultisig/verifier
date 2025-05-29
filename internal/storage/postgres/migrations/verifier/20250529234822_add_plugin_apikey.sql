-- +goose Up
-- +goose StatementBegin
CREATE TABLE plugin_apikey (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    plugin_id plugin_id NOT NULL REFERENCES plugins(id) ON DELETE CASCADE,
    apikey TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NULL,
    status INT NOT NULL DEFAULT 1 -- 1: active, 0: inactive
);
CREATE INDEX idx_plugin_apikey_plugin_id ON plugin_apikey(plugin_id);
CREATE INDEX idx_plugin_apikey_apikey ON plugin_apikey(apikey);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ROLLBACK;
-- +goose StatementEnd
