-- +goose Up
ALTER TABLE plugin_policies ADD COLUMN deactivation_reason TEXT;

-- +goose Down
ALTER TABLE plugin_policies DROP COLUMN deactivation_reason;
