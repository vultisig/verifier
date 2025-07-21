-- +goose Up
-- +goose StatementBegin
ALTER TABLE plugin_policies ADD COLUMN IF NOT EXISTS deleted boolean NOT NULL DEFAULT false;

-- Ensure active is set to false when deleted is set to true
CREATE OR REPLACE FUNCTION set_policy_inactive_on_delete()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.deleted = true THEN
        NEW.active := false;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_set_policy_inactive_on_delete ON plugin_policies;
CREATE TRIGGER trg_set_policy_inactive_on_delete
    BEFORE UPDATE ON plugin_policies
    FOR EACH ROW
    WHEN (OLD.deleted = false AND NEW.deleted = true)
    EXECUTE FUNCTION set_policy_inactive_on_delete();

-- Prevent updates to plugin_policies if deleted=true
CREATE OR REPLACE FUNCTION prevent_update_if_policy_deleted()
RETURNS TRIGGER AS $$
BEGIN
    IF OLD.deleted = true THEN
        RAISE EXCEPTION 'Cannot update a deleted policy';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_prevent_update_if_policy_deleted ON plugin_policies;
CREATE TRIGGER trg_prevent_update_if_policy_deleted
    BEFORE UPDATE ON plugin_policies
    FOR EACH ROW
    WHEN (OLD.deleted = true)
    EXECUTE FUNCTION prevent_update_if_policy_deleted();

-- Prevent changes to plugin_policy_billing if parent policy is deleted
CREATE OR REPLACE FUNCTION prevent_billing_update_if_policy_deleted()
RETURNS TRIGGER AS $$
DECLARE
    is_deleted boolean;
BEGIN
    SELECT deleted INTO is_deleted FROM plugin_policies WHERE id = NEW.plugin_policy_id;
    IF is_deleted THEN
        RAISE EXCEPTION 'Cannot modify billing for a deleted policy';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_prevent_billing_update_if_policy_deleted ON plugin_policy_billing;
CREATE TRIGGER trg_prevent_billing_update_if_policy_deleted
    BEFORE INSERT OR UPDATE OR DELETE ON plugin_policy_billing
    FOR EACH ROW
    EXECUTE FUNCTION prevent_billing_update_if_policy_deleted();

-- Prevent changes to fees if parent policy is deleted
CREATE OR REPLACE FUNCTION prevent_fees_update_if_policy_deleted()
RETURNS TRIGGER AS $$
DECLARE
    is_deleted boolean;
BEGIN
    SELECT p.deleted INTO is_deleted
    FROM plugin_policies p
    JOIN plugin_policy_billing b ON b.plugin_policy_id = p.id
    WHERE b.id = NEW.plugin_policy_billing_id;
    IF is_deleted THEN
        RAISE EXCEPTION 'Cannot modify fees for a deleted policy';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_prevent_fees_update_if_policy_deleted ON fees;
CREATE TRIGGER trg_prevent_fees_update_if_policy_deleted
    BEFORE INSERT OR UPDATE OR DELETE ON fees
    FOR EACH ROW
    EXECUTE FUNCTION prevent_fees_update_if_policy_deleted();

-- Prevent changes to plugin_policy_sync if parent policy is deleted
CREATE OR REPLACE FUNCTION prevent_sync_update_if_policy_deleted()
RETURNS TRIGGER AS $$
DECLARE
    is_deleted boolean;
BEGIN
    SELECT deleted INTO is_deleted FROM plugin_policies WHERE id = NEW.policy_id;
    IF is_deleted THEN
        RAISE EXCEPTION 'Cannot modify sync for a deleted policy';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_prevent_sync_update_if_policy_deleted ON plugin_policy_sync;
CREATE TRIGGER trg_prevent_sync_update_if_policy_deleted
    BEFORE INSERT OR UPDATE OR DELETE ON plugin_policy_sync
    FOR EACH ROW
    EXECUTE FUNCTION prevent_sync_update_if_policy_deleted();

-- Prevent changes to tx_indexer if parent policy is deleted
CREATE OR REPLACE FUNCTION prevent_tx_indexer_update_if_policy_deleted()
RETURNS TRIGGER AS $$
DECLARE
    is_deleted boolean;
BEGIN
    SELECT deleted INTO is_deleted FROM plugin_policies WHERE id = NEW.policy_id;
    IF is_deleted THEN
        RAISE EXCEPTION 'Cannot modify tx_indexer for a deleted policy';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_prevent_tx_indexer_update_if_policy_deleted ON tx_indexer;
CREATE TRIGGER trg_prevent_tx_indexer_update_if_policy_deleted
    BEFORE INSERT OR UPDATE OR DELETE ON tx_indexer
    FOR EACH ROW
    EXECUTE FUNCTION prevent_tx_indexer_update_if_policy_deleted();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE plugin_policies DROP COLUMN IF EXISTS deleted;
DROP TRIGGER IF EXISTS trg_set_policy_inactive_on_delete ON plugin_policies;
DROP FUNCTION IF EXISTS set_policy_inactive_on_delete();
DROP TRIGGER IF EXISTS trg_prevent_update_if_policy_deleted ON plugin_policies;
DROP FUNCTION IF EXISTS prevent_update_if_policy_deleted();
DROP TRIGGER IF EXISTS trg_prevent_billing_update_if_policy_deleted ON plugin_policy_billing;
DROP FUNCTION IF EXISTS prevent_billing_update_if_policy_deleted();
DROP TRIGGER IF EXISTS trg_prevent_fees_update_if_policy_deleted ON fees;
DROP FUNCTION IF EXISTS prevent_fees_update_if_policy_deleted();
DROP TRIGGER IF EXISTS trg_prevent_sync_update_if_policy_deleted ON plugin_policy_sync;
DROP FUNCTION IF EXISTS prevent_sync_update_if_policy_deleted();
DROP TRIGGER IF EXISTS trg_prevent_tx_indexer_update_if_policy_deleted ON tx_indexer;
DROP FUNCTION IF EXISTS prevent_tx_indexer_update_if_policy_deleted();
-- +goose StatementEnd
