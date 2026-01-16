-- +goose Up
-- +goose StatementBegin
ALTER TABLE plugin_policies ADD COLUMN deactivation_reason TEXT DEFAULT NULL;
-- +goose StatementEnd

-- +goose Down
