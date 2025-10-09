-- +goose Up
-- +goose StatementBegin

-- Change plugin_id from ENUM to VARCHAR by creating a domain type that aliases to VARCHAR
-- This approach avoids having to drop and recreate all columns and dependencies

-- Drop the old enum type and recreate as a domain type aliasing to VARCHAR
DROP TYPE plugin_id CASCADE;
CREATE DOMAIN plugin_id AS VARCHAR(255);

-- Clear existing data since columns were dropped by CASCADE
-- Data will be resynced from proposed.yaml on service start
TRUNCATE TABLE plugins CASCADE;

-- Recreate the id column in plugins table as first column
ALTER TABLE plugins ADD COLUMN id plugin_id NOT NULL;
ALTER TABLE plugins ALTER COLUMN id SET STORAGE MAIN;

-- Reorder columns to put id first by recreating the table
CREATE TABLE plugins_new (
    id plugin_id NOT NULL,
    title VARCHAR(255) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    server_endpoint TEXT NOT NULL,
    category plugin_category NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id)
);

-- Copy data from old table
INSERT INTO plugins_new (id, title, description, server_endpoint, category, created_at, updated_at)
SELECT id, title, description, server_endpoint, category, created_at, updated_at FROM plugins;

-- Drop old table and rename new one
DROP TABLE plugins;
ALTER TABLE plugins_new RENAME TO plugins;

-- Recreate foreign key columns that were dropped by CASCADE
-- Using nullable first since tables might have data, then set NOT NULL
ALTER TABLE plugin_apikey ADD COLUMN plugin_id plugin_id;
ALTER TABLE plugin_policies ADD COLUMN plugin_id plugin_id;
ALTER TABLE plugin_policy_sync ADD COLUMN plugin_id plugin_id;
ALTER TABLE plugin_ratings ADD COLUMN plugin_id plugin_id;
ALTER TABLE plugin_tags ADD COLUMN plugin_id plugin_id;
ALTER TABLE pricings ADD COLUMN plugin_id plugin_id;
ALTER TABLE reviews ADD COLUMN plugin_id plugin_id;

-- Clear any remaining data in related tables
TRUNCATE TABLE plugin_apikey CASCADE;
TRUNCATE TABLE plugin_policies CASCADE;
TRUNCATE TABLE plugin_policy_sync CASCADE;
TRUNCATE TABLE plugin_ratings CASCADE;
TRUNCATE TABLE plugin_tags CASCADE;
TRUNCATE TABLE pricings CASCADE;
TRUNCATE TABLE reviews CASCADE;

-- Now set NOT NULL constraints
ALTER TABLE plugin_apikey ALTER COLUMN plugin_id SET NOT NULL;
ALTER TABLE plugin_policies ALTER COLUMN plugin_id SET NOT NULL;
ALTER TABLE plugin_policy_sync ALTER COLUMN plugin_id SET NOT NULL;
ALTER TABLE plugin_ratings ALTER COLUMN plugin_id SET NOT NULL;
ALTER TABLE plugin_tags ALTER COLUMN plugin_id SET NOT NULL;
ALTER TABLE pricings ALTER COLUMN plugin_id SET NOT NULL;

-- Recreate foreign key constraints
ALTER TABLE plugin_apikey ADD CONSTRAINT plugin_apikey_plugin_id_fkey
    FOREIGN KEY (plugin_id) REFERENCES plugins(id) ON DELETE CASCADE;
ALTER TABLE plugin_policy_sync ADD CONSTRAINT plugin_policy_sync_plugin_id_fkey
    FOREIGN KEY (plugin_id) REFERENCES plugins(id) ON DELETE CASCADE;
ALTER TABLE plugin_ratings ADD CONSTRAINT plugin_ratings_plugin_id_fkey
    FOREIGN KEY (plugin_id) REFERENCES plugins(id) ON DELETE CASCADE;
ALTER TABLE plugin_tags ADD CONSTRAINT plugin_tags_plugin_id_fkey
    FOREIGN KEY (plugin_id) REFERENCES plugins(id) ON DELETE CASCADE;
ALTER TABLE pricings ADD CONSTRAINT pricings_plugin_id_fkey
    FOREIGN KEY (plugin_id) REFERENCES plugins(id) ON DELETE CASCADE;
ALTER TABLE reviews ADD CONSTRAINT reviews_plugin_id_fkey
    FOREIGN KEY (plugin_id) REFERENCES plugins(id) ON DELETE CASCADE;

-- Recreate indexes
CREATE INDEX idx_plugin_apikey_plugin_id ON plugin_apikey (plugin_id);
CREATE INDEX idx_plugin_policies_plugin_id ON plugin_policies (plugin_id);
CREATE INDEX idx_reviews_plugin_id ON reviews (plugin_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- +goose StatementEnd
