-- +goose Up
-- +goose StatementBegin

-- Rebuild PRIMARY KEY and FOREIGN KEY constraints on verifier tables after
-- the system migration converted plugin_id from ENUM to TEXT/DOMAIN.
--
-- The system migration dynamically drops all constraints on plugin_id columns
-- but only rebuilds the unique_fees_policy_per_public_key index on
-- plugin_policies (the only table guaranteed to exist in system migrations).
--
-- This migration is idempotent: on fresh installs the constraints already
-- exist from the original CREATE TABLE statements, so we skip them.

DO $$
BEGIN
    -- ============================================================
    -- PRIMARY KEYS
    -- ============================================================

    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'plugins_pkey') THEN
        ALTER TABLE plugins ADD CONSTRAINT plugins_pkey PRIMARY KEY (id);
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'plugin_installations_pkey') THEN
        ALTER TABLE plugin_installations ADD CONSTRAINT plugin_installations_pkey PRIMARY KEY (plugin_id, public_key);
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'plugin_owners_pkey') THEN
        ALTER TABLE plugin_owners ADD CONSTRAINT plugin_owners_pkey PRIMARY KEY (plugin_id, public_key);
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'plugin_ratings_pkey') THEN
        ALTER TABLE plugin_ratings ADD CONSTRAINT plugin_ratings_pkey PRIMARY KEY (plugin_id);
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'plugin_reports_pkey') THEN
        ALTER TABLE plugin_reports ADD CONSTRAINT plugin_reports_pkey PRIMARY KEY (plugin_id, reporter_public_key);
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'plugin_tags_pkey') THEN
        ALTER TABLE plugin_tags ADD CONSTRAINT plugin_tags_pkey PRIMARY KEY (plugin_id, tag_id);
    END IF;

    -- ============================================================
    -- FOREIGN KEYS
    -- ============================================================

    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'plugin_apikey_plugin_id_fkey') THEN
        ALTER TABLE plugin_apikey ADD CONSTRAINT plugin_apikey_plugin_id_fkey
            FOREIGN KEY (plugin_id) REFERENCES plugins(id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'plugin_images_plugin_id_fkey') THEN
        ALTER TABLE plugin_images ADD CONSTRAINT plugin_images_plugin_id_fkey
            FOREIGN KEY (plugin_id) REFERENCES plugins(id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'plugin_owners_plugin_id_fkey') THEN
        ALTER TABLE plugin_owners ADD CONSTRAINT plugin_owners_plugin_id_fkey
            FOREIGN KEY (plugin_id) REFERENCES plugins(id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'plugin_pause_history_plugin_id_fkey') THEN
        ALTER TABLE plugin_pause_history ADD CONSTRAINT plugin_pause_history_plugin_id_fkey
            FOREIGN KEY (plugin_id) REFERENCES plugins(id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'plugin_policy_sync_plugin_id_fkey') THEN
        ALTER TABLE plugin_policy_sync ADD CONSTRAINT plugin_policy_sync_plugin_id_fkey
            FOREIGN KEY (plugin_id) REFERENCES plugins(id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'plugin_ratings_plugin_id_fkey') THEN
        ALTER TABLE plugin_ratings ADD CONSTRAINT plugin_ratings_plugin_id_fkey
            FOREIGN KEY (plugin_id) REFERENCES plugins(id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'plugin_reports_plugin_id_fkey') THEN
        ALTER TABLE plugin_reports ADD CONSTRAINT plugin_reports_plugin_id_fkey
            FOREIGN KEY (plugin_id) REFERENCES plugins(id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'plugin_tags_plugin_id_fkey') THEN
        ALTER TABLE plugin_tags ADD CONSTRAINT plugin_tags_plugin_id_fkey
            FOREIGN KEY (plugin_id) REFERENCES plugins(id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'pricings_plugin_id_fkey') THEN
        ALTER TABLE pricings ADD CONSTRAINT pricings_plugin_id_fkey
            FOREIGN KEY (plugin_id) REFERENCES plugins(id) ON DELETE CASCADE;
    END IF;

    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'reviews_plugin_id_fkey') THEN
        ALTER TABLE reviews ADD CONSTRAINT reviews_plugin_id_fkey
            FOREIGN KEY (plugin_id) REFERENCES plugins(id) ON DELETE CASCADE;
    END IF;
END;
$$;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Drop FKs
ALTER TABLE plugin_apikey        DROP CONSTRAINT IF EXISTS plugin_apikey_plugin_id_fkey;
ALTER TABLE plugin_images        DROP CONSTRAINT IF EXISTS plugin_images_plugin_id_fkey;
ALTER TABLE plugin_owners        DROP CONSTRAINT IF EXISTS plugin_owners_plugin_id_fkey;
ALTER TABLE plugin_pause_history DROP CONSTRAINT IF EXISTS plugin_pause_history_plugin_id_fkey;
ALTER TABLE plugin_policy_sync   DROP CONSTRAINT IF EXISTS plugin_policy_sync_plugin_id_fkey;
ALTER TABLE plugin_ratings       DROP CONSTRAINT IF EXISTS plugin_ratings_plugin_id_fkey;
ALTER TABLE plugin_reports       DROP CONSTRAINT IF EXISTS plugin_reports_plugin_id_fkey;
ALTER TABLE plugin_tags          DROP CONSTRAINT IF EXISTS plugin_tags_plugin_id_fkey;
ALTER TABLE pricings             DROP CONSTRAINT IF EXISTS pricings_plugin_id_fkey;
ALTER TABLE reviews              DROP CONSTRAINT IF EXISTS reviews_plugin_id_fkey;

-- Drop PKs
ALTER TABLE plugins              DROP CONSTRAINT IF EXISTS plugins_pkey;
ALTER TABLE plugin_installations DROP CONSTRAINT IF EXISTS plugin_installations_pkey;
ALTER TABLE plugin_owners        DROP CONSTRAINT IF EXISTS plugin_owners_pkey;
ALTER TABLE plugin_ratings       DROP CONSTRAINT IF EXISTS plugin_ratings_pkey;
ALTER TABLE plugin_reports       DROP CONSTRAINT IF EXISTS plugin_reports_pkey;
ALTER TABLE plugin_tags          DROP CONSTRAINT IF EXISTS plugin_tags_pkey;

-- +goose StatementEnd
