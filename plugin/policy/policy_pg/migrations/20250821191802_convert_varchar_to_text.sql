-- +goose Up
-- +goose StatementBegin
ALTER TABLE "plugin_policies"
    ALTER COLUMN public_key TYPE TEXT,
    ALTER COLUMN plugin_id TYPE TEXT,
    ALTER COLUMN plugin_version TYPE TEXT,
    ALTER COLUMN signature TYPE TEXT,
    ALTER COLUMN recipe TYPE TEXT;
-- +goose StatementEnd
