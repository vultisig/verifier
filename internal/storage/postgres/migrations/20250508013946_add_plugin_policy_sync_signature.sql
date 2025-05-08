-- +goose Up
-- +goose StatementBegin
ALTER TABLE plugin_policy_sync
    ADD COLUMN signature TEXT DEFAULT TRUE;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE plugin_policy_sync DROP COLUMN signature;
-- +goose StatementEnd
