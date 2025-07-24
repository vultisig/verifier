-- +goose Up
-- +goose StatementBegin

-- DB level constraints to prevent changes to plugin_policy_billing, fees, and tx_indexer if the parent policy is deleted

-- Prevent changes to plugin_policy_billing if parent policy is deleted
CREATE OR REPLACE FUNCTION prevent_billing_update_if_policy_deleted()
RETURNS TRIGGER AS $$
DECLARE
    is_deleted boolean;
BEGIN
    SELECT deleted INTO is_deleted FROM plugin_policies WHERE id = COALESCE(NEW.plugin_policy_id, OLD.plugin_policy_id);
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
    WHERE b.id = COALESCE(NEW.plugin_policy_billing_id, OLD.plugin_policy_billing_id);
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

-- Prevent changes to tx_indexer if parent policy is deleted
CREATE OR REPLACE FUNCTION prevent_tx_indexer_update_if_policy_deleted()
RETURNS TRIGGER AS $$
DECLARE
    is_deleted boolean;
BEGIN
    SELECT deleted INTO is_deleted FROM plugin_policies WHERE id = COALESCE(NEW.policy_id, OLD.policy_id);
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
DROP TRIGGER IF EXISTS trg_prevent_billing_update_if_policy_deleted ON plugin_policy_billing;
DROP FUNCTION IF EXISTS prevent_billing_update_if_policy_deleted();
DROP TRIGGER IF EXISTS trg_prevent_fees_update_if_policy_deleted ON fees;
DROP FUNCTION IF EXISTS prevent_fees_update_if_policy_deleted();
DROP TRIGGER IF EXISTS trg_prevent_sync_update_if_policy_deleted ON plugin_policy_sync;
DROP FUNCTION IF EXISTS prevent_sync_update_if_policy_deleted();
DROP TRIGGER IF EXISTS trg_prevent_tx_indexer_update_if_policy_deleted ON tx_indexer;
DROP FUNCTION IF EXISTS prevent_tx_indexer_update_if_policy_deleted();
-- +goose StatementEnd
