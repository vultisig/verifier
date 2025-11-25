-- +goose Up
ALTER TABLE plugins
    ADD COLUMN thumbnail_url TEXT NOT NULL DEFAULT '',
    ADD COLUMN images JSONB NOT NULL DEFAULT '[]';

-- +goose Down
ALTER TABLE plugins
    DROP COLUMN images,
    DROP COLUMN thumbnail_url;
