-- +goose Up
-- +goose StatementBegin
ALTER TABLE plugin_policies DROP COLUMN derive_path;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE plugin_policies ADD COLUMN derive_path TEXT NOT NULL;
-- +goose StatementEnd
