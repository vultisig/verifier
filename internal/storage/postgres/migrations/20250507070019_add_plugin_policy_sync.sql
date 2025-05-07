-- +goose Up
-- +goose StatementBegin
CREATE TABLE plugin_policy_sync (
    id UUID PRIMARY KEY,
    policy_id UUID NOT NULL,
    sync_type INT NOT NULL,
    status INT NOT NULL,
    reason TEXT,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
-- Index for faster lookups on policy_id
CREATE INDEX idx_plugin_policy_sync_policy_id ON plugin_policy_sync(policy_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS plugin_policy_sync;
DROP INDEX IF EXISTS idx_plugin_policy_sync_policy_id;
-- +goose StatementEnd
