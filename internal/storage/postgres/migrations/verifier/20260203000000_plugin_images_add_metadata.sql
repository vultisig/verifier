-- +goose Up
ALTER TABLE plugin_images
  ADD COLUMN content_type TEXT NOT NULL,
  ADD COLUMN filename TEXT NOT NULL;

ALTER TABLE plugin_images
  ADD CONSTRAINT plugin_images_content_type_check
  CHECK (content_type IN ('image/jpeg','image/png','image/webp'));

-- +goose Down
ALTER TABLE plugin_images
  DROP CONSTRAINT IF EXISTS plugin_images_content_type_check;

ALTER TABLE plugin_images
  DROP COLUMN IF EXISTS content_type,
  DROP COLUMN IF EXISTS filename;
