-- +goose Up

ALTER TABLE plugin_reports ADD COLUMN details TEXT;

-- +goose Down

ALTER TABLE plugin_reports DROP COLUMN IF EXISTS details;
