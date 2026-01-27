-- +goose Up

ALTER TABLE plugin_reports ADD COLUMN details TEXT NOT NULL DEFAULT '';

-- +goose Down

ALTER TABLE plugin_reports DROP COLUMN IF EXISTS details;
