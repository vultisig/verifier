-- +goose Up
-- +goose StatementBegin
ALTER TYPE plugin_category ADD VALUE IF NOT EXISTS 'app';
-- +goose StatementEnd

-- +goose Down
-- Cannot remove enum values in PostgreSQL
