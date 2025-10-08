-- +goose Up
-- +goose StatementBegin

BEGIN;

-- Step 1: Add temporary columns for all tables referencing plugin_id
ALTER TABLE plugin_apikey ADD COLUMN plugin_id_temp VARCHAR(255);
ALTER TABLE plugin_policy_sync ADD COLUMN plugin_id_temp VARCHAR(255);
ALTER TABLE plugin_ratings ADD COLUMN plugin_id_temp VARCHAR(255);
ALTER TABLE plugin_tags ADD COLUMN plugin_id_temp VARCHAR(255);
ALTER TABLE plugins ADD COLUMN id_temp VARCHAR(255);
ALTER TABLE pricings ADD COLUMN plugin_id_temp VARCHAR(255);
ALTER TABLE reviews ADD COLUMN plugin_id_temp VARCHAR(255);

-- Step 2: Copy data from enum columns to temp varchar columns
UPDATE plugin_apikey SET plugin_id_temp = plugin_id::text;
UPDATE plugin_policy_sync SET plugin_id_temp = plugin_id::text;
UPDATE plugin_ratings SET plugin_id_temp = plugin_id::text;
UPDATE plugin_tags SET plugin_id_temp = plugin_id::text;
UPDATE plugins SET id_temp = id::text;
UPDATE pricings SET plugin_id_temp = plugin_id::text;
UPDATE reviews SET plugin_id_temp = plugin_id::text;

-- Step 3: Drop foreign key constraints
ALTER TABLE plugin_apikey DROP CONSTRAINT plugin_apikey_plugin_id_fkey;
ALTER TABLE plugin_policy_sync DROP CONSTRAINT plugin_policy_sync_plugin_id_fkey;
ALTER TABLE plugin_ratings DROP CONSTRAINT plugin_ratings_plugin_id_fkey;
ALTER TABLE plugin_tags DROP CONSTRAINT plugin_tags_plugin_id_fkey;
ALTER TABLE pricings DROP CONSTRAINT pricings_plugin_id_fkey;
ALTER TABLE reviews DROP CONSTRAINT reviews_plugin_id_fkey;

-- Step 4: Drop old enum columns
ALTER TABLE plugin_apikey DROP COLUMN plugin_id;
ALTER TABLE plugin_policy_sync DROP COLUMN plugin_id;
ALTER TABLE plugin_ratings DROP COLUMN plugin_id;
ALTER TABLE plugin_tags DROP COLUMN plugin_id;
ALTER TABLE plugins DROP COLUMN id;
ALTER TABLE pricings DROP COLUMN plugin_id;
ALTER TABLE reviews DROP COLUMN plugin_id;

-- Step 5: Rename temp columns to original names
ALTER TABLE plugin_apikey RENAME COLUMN plugin_id_temp TO plugin_id;
ALTER TABLE plugin_policy_sync RENAME COLUMN plugin_id_temp TO plugin_id;
ALTER TABLE plugin_ratings RENAME COLUMN plugin_id_temp TO plugin_id;
ALTER TABLE plugin_tags RENAME COLUMN plugin_id_temp TO plugin_id;
ALTER TABLE plugins RENAME COLUMN id_temp TO id;
ALTER TABLE pricings RENAME COLUMN plugin_id_temp TO plugin_id;
ALTER TABLE reviews RENAME COLUMN plugin_id_temp TO plugin_id;

-- Step 6: Set NOT NULL constraints
ALTER TABLE plugin_apikey ALTER COLUMN plugin_id SET NOT NULL;
ALTER TABLE plugin_policy_sync ALTER COLUMN plugin_id SET NOT NULL;
ALTER TABLE plugin_ratings ALTER COLUMN plugin_id SET NOT NULL;
ALTER TABLE plugin_tags ALTER COLUMN plugin_id SET NOT NULL;
ALTER TABLE plugins ALTER COLUMN id SET NOT NULL;
ALTER TABLE pricings ALTER COLUMN plugin_id SET NOT NULL;

-- Step 7: Add primary key back to plugins
ALTER TABLE plugins ADD PRIMARY KEY (id);

-- Step 8: Recreate foreign key constraints
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

-- Step 9: Drop the plugin_id enum type
DROP TYPE plugin_id;

COMMIT;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- +goose StatementEnd
