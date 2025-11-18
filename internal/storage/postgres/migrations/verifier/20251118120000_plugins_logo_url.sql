-- +goose Up
ALTER TABLE plugins
ADD COLUMN logo_url TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE plugins
DROP COLUMN logo_url;
