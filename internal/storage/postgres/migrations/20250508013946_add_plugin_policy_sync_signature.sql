-- +goose Up
-- +goose StatementBegin
ALTER TABLE plugin_policy_sync
    ADD COLUMN signature TEXT;
ALTER TABLE plugin_policy_sync
    ADD COLUMN  plugin_id uuid NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE plugin_policy_sync DROP COLUMN signature;
ALTER TABLE plugin_policy_sync DROP COLUMN plugin_id;
-- +goose StatementEnd
