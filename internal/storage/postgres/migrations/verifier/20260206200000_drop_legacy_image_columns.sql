-- +goose Up
ALTER TABLE plugins DROP COLUMN IF EXISTS logo_url;
ALTER TABLE plugins DROP COLUMN IF EXISTS thumbnail_url;
ALTER TABLE plugins DROP COLUMN IF EXISTS images;

-- +goose Down
