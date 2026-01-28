-- +goose Up
CREATE TABLE plugin_images (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    plugin_id plugin_id NOT NULL REFERENCES plugins(id) ON DELETE CASCADE,
    image_type TEXT NOT NULL CHECK (image_type IN ('logo', 'banner', 'thumbnail', 'media')),
    s3_path TEXT NOT NULL,
    image_order INTEGER NOT NULL DEFAULT 0,
    uploaded_by_public_key TEXT NOT NULL,
    visible BOOLEAN NOT NULL DEFAULT true,
    deleted BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_plugin_images_plugin_id ON plugin_images(plugin_id);
CREATE INDEX idx_plugin_images_plugin_type
  ON plugin_images(plugin_id, image_type)
  WHERE deleted = false AND visible = true;

CREATE UNIQUE INDEX idx_plugin_images_logo_unique
  ON plugin_images(plugin_id) WHERE image_type = 'logo' AND deleted = false AND visible = true;
CREATE UNIQUE INDEX idx_plugin_images_thumbnail_unique
  ON plugin_images(plugin_id) WHERE image_type = 'thumbnail' AND deleted = false AND visible = true;
CREATE UNIQUE INDEX idx_plugin_images_banner_unique
  ON plugin_images(plugin_id) WHERE image_type = 'banner' AND deleted = false AND visible = true;

CREATE UNIQUE INDEX idx_plugin_images_media_order_unique
  ON plugin_images(plugin_id, image_order) WHERE image_type = 'media' AND deleted = false AND visible = true;

-- +goose Down
DROP TABLE IF EXISTS plugin_images;
