-- +goose Up
ALTER TABLE plugins
    ADD COLUMN logo_s3_key TEXT DEFAULT '' NOT NULL,
    ADD COLUMN thumbnail_s3_key TEXT DEFAULT '' NOT NULL;

-- +goose Down
ALTER TABLE plugins
    DROP COLUMN IF EXISTS logo_s3_key,
    DROP COLUMN IF EXISTS thumbnail_s3_key;
