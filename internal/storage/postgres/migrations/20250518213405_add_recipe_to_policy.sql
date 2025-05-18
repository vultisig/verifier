-- +goose Up
-- +goose StatementBegin
ALTER TABLE plugin_policies
    ADD COLUMN recipe TEXT NOT NULL DEFAULT '';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE plugin_policies
    DROP COLUMN recipe;
-- +goose StatementEnd
