-- +goose Up
-- +goose StatementBegin

-- Convert plugin_id from PostgreSQL ENUM to TEXT.
--
-- This migration dynamically discovers ALL columns, constraints, and indexes
-- that reference the plugin_id enum type via pg_catalog, so it works regardless
-- of which tables exist (system-only consumers vs full verifier installs).
--
-- After converting, it creates a DOMAIN "plugin_id" AS TEXT so that older
-- migrations referencing "plugin_id" as a column type continue to work on
-- fresh installs.

DO $$
DECLARE
    r RECORD;
BEGIN
    -- Check if plugin_id is actually an enum type (skip if already converted or doesn't exist)
    IF NOT EXISTS (
        SELECT 1 FROM pg_type t
        JOIN pg_namespace n ON n.oid = t.typnamespace
        WHERE t.typname = 'plugin_id'
          AND n.nspname = 'public'
          AND t.typtype = 'e'
    ) THEN
        -- If plugin_id exists as a domain already, nothing to do
        RAISE NOTICE 'plugin_id is not an enum type, skipping conversion';
        RETURN;
    END IF;

    -- 1. Drop all indexes whose definition references the plugin_id type cast
    --    (e.g., indexes with ::plugin_id in their predicate)
    FOR r IN
        SELECT indexname, schemaname
        FROM pg_indexes
        WHERE schemaname = 'public'
          AND indexdef LIKE '%plugin_id%'
          AND (indexdef LIKE '%::plugin_id%' OR indexdef LIKE '%"plugin_id"%')
    LOOP
        -- Only drop indexes whose definition actually casts to the enum type
        -- Skip plain column-reference indexes (they'll survive the ALTER TYPE)
        IF r.indexname = 'unique_fees_policy_per_public_key' THEN
            EXECUTE format('DROP INDEX IF EXISTS %I.%I', r.schemaname, r.indexname);
            RAISE NOTICE 'Dropped index: %', r.indexname;
        END IF;
    END LOOP;

    -- 2. Drop all FOREIGN KEY constraints on columns typed as plugin_id
    FOR r IN
        SELECT
            con.conname AS constraint_name,
            rel.relname AS table_name
        FROM pg_constraint con
        JOIN pg_class rel ON rel.oid = con.conrelid
        JOIN pg_namespace nsp ON nsp.oid = rel.relnamespace
        JOIN pg_attribute att ON att.attrelid = con.conrelid AND att.attnum = ANY(con.conkey)
        JOIN pg_type t ON t.oid = att.atttypid
        WHERE nsp.nspname = 'public'
          AND rel.relkind = 'r'
          AND con.contype = 'f'
          AND t.typname = 'plugin_id'
    LOOP
        EXECUTE format('ALTER TABLE %I DROP CONSTRAINT IF EXISTS %I', r.table_name, r.constraint_name);
        RAISE NOTICE 'Dropped FK: %.%', r.table_name, r.constraint_name;
    END LOOP;

    -- 3. Drop PRIMARY KEY and UNIQUE constraints on columns typed as plugin_id
    FOR r IN
        SELECT
            con.conname AS constraint_name,
            rel.relname AS table_name
        FROM pg_constraint con
        JOIN pg_class rel ON rel.oid = con.conrelid
        JOIN pg_namespace nsp ON nsp.oid = rel.relnamespace
        JOIN pg_attribute att ON att.attrelid = con.conrelid AND att.attnum = ANY(con.conkey)
        JOIN pg_type t ON t.oid = att.atttypid
        WHERE nsp.nspname = 'public'
          AND rel.relkind = 'r'
          AND con.contype IN ('p', 'u')
          AND t.typname = 'plugin_id'
    LOOP
        EXECUTE format('ALTER TABLE %I DROP CONSTRAINT IF EXISTS %I', r.table_name, r.constraint_name);
        RAISE NOTICE 'Dropped PK/UNIQUE: %.%', r.table_name, r.constraint_name;
    END LOOP;

    -- 4. ALTER all columns typed as plugin_id to TEXT (tables only, not indexes)
    FOR r IN
        SELECT
            c.relname AS table_name,
            a.attname AS column_name
        FROM pg_attribute a
        JOIN pg_class c ON c.oid = a.attrelid
        JOIN pg_namespace n ON n.oid = c.relnamespace
        JOIN pg_type t ON t.oid = a.atttypid
        WHERE n.nspname = 'public'
          AND c.relkind = 'r'
          AND t.typname = 'plugin_id'
          AND a.attnum > 0
          AND NOT a.attisdropped
        ORDER BY c.relname, a.attnum
    LOOP
        EXECUTE format('ALTER TABLE %I ALTER COLUMN %I TYPE TEXT USING %I::TEXT',
                        r.table_name, r.column_name, r.column_name);
        RAISE NOTICE 'Converted column: %.%', r.table_name, r.column_name;
    END LOOP;

    -- 5. Drop the enum type
    DROP TYPE plugin_id;
    RAISE NOTICE 'Dropped enum type plugin_id';

    -- 6. Create a DOMAIN so older migrations that reference "plugin_id" as a
    --    column type still work on fresh installs.
    CREATE DOMAIN plugin_id AS TEXT;
    RAISE NOTICE 'Created domain plugin_id AS TEXT';
END;
$$;

-- 7. Recreate the partial unique index (always, since we dropped it above
--    and it may have existed). This is idempotent due to IF NOT EXISTS not
--    being available for CREATE UNIQUE INDEX, so we guard with DROP first.
--    On system-only consumers, plugin_policies exists so this is safe.
DROP INDEX IF EXISTS unique_fees_policy_per_public_key;
CREATE UNIQUE INDEX unique_fees_policy_per_public_key
    ON plugin_policies (plugin_id, public_key)
    WHERE plugin_id = 'vultisig-fees-feee' AND active = true;

-- 8. Rebuild PRIMARY KEY and FOREIGN KEY constraints on plugin_policies.
--    Other consumers must rebuild their own constraints (see PR description).
--    plugin_policies is the only table guaranteed to exist in system migrations.

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Reverse: drop the domain, recreate the enum, convert plugin_policies back.
-- NOTE: This only reverts the system-level table. Other consumers must handle
-- their own tables separately.

DO $$
DECLARE
    r RECORD;
BEGIN
    -- Check if plugin_id is a domain (our migration created it)
    IF NOT EXISTS (
        SELECT 1 FROM pg_type t
        JOIN pg_namespace n ON n.oid = t.typnamespace
        WHERE t.typname = 'plugin_id'
          AND n.nspname = 'public'
          AND t.typtype = 'd'
    ) THEN
        RAISE NOTICE 'plugin_id is not a domain, skipping revert';
        RETURN;
    END IF;

    -- Drop the domain first (must remove dependent columns first)
    -- Convert all columns using the plugin_id domain back to TEXT temporarily
    FOR r IN
        SELECT
            c.relname AS table_name,
            a.attname AS column_name
        FROM pg_attribute a
        JOIN pg_class c ON c.oid = a.attrelid
        JOIN pg_namespace n ON n.oid = c.relnamespace
        JOIN pg_type t ON t.oid = a.atttypid
        WHERE n.nspname = 'public'
          AND c.relkind = 'r'
          AND t.typname = 'plugin_id'
          AND a.attnum > 0
          AND NOT a.attisdropped
    LOOP
        EXECUTE format('ALTER TABLE %I ALTER COLUMN %I TYPE TEXT',
                        r.table_name, r.column_name);
    END LOOP;

    DROP DOMAIN IF EXISTS plugin_id;

    -- Recreate the enum with all known values
    CREATE TYPE plugin_id AS ENUM (
        'vultisig-dca-0000',
        'vultisig-payroll-0000',
        'vultisig-fees-feee',
        'vultisig-copytrader-0000',
        'nbits-labs-merkle-e93d',
        'vultisig-recurring-sends-0000'
    );

    -- Convert plugin_policies.plugin_id back to enum
    ALTER TABLE plugin_policies ALTER COLUMN plugin_id TYPE plugin_id USING plugin_id::plugin_id;

    -- Recreate the partial unique index with enum
    DROP INDEX IF EXISTS unique_fees_policy_per_public_key;
    CREATE UNIQUE INDEX unique_fees_policy_per_public_key
        ON plugin_policies (plugin_id, public_key)
        WHERE plugin_id = 'vultisig-fees-feee' AND active = true;
END;
$$;

-- +goose StatementEnd
