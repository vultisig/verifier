-- +goose Up
ALTER TABLE plugins
ADD COLUMN logo_url TEXT;

-- +goose Down
ALTER TABLE plugins
DROP COLUMN logo_url;