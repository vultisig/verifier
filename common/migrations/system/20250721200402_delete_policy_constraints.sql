-- +goose Up
-- +goose StatementBegin
ALTER TABLE plugin_policies ADD COLUMN IF NOT EXISTS deleted boolean NOT NULL DEFAULT false;



-- Sets the active to false when the policy is deleted
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
    BEFORE INSERT OR UPDATE ON plugin_policies
    FOR EACH ROW
    WHEN (NEW.deleted = true)
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



-- Prevent inserts of a deleted plugin policy
CREATE OR REPLACE FUNCTION prevent_insert_if_policy_deleted()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.deleted = true THEN
        RAISE EXCEPTION 'Cannot insert a deleted policy';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_prevent_insert_if_policy_deleted ON plugin_policies;
CREATE TRIGGER trg_prevent_insert_if_policy_deleted
    BEFORE INSERT ON plugin_policies    
    FOR EACH ROW
    EXECUTE FUNCTION prevent_insert_if_policy_deleted();



-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS trg_prevent_insert_if_policy_deleted ON plugin_policies;
DROP FUNCTION IF EXISTS prevent_insert_if_policy_deleted();
DROP TRIGGER IF EXISTS trg_prevent_update_if_policy_deleted ON plugin_policies;
DROP FUNCTION IF EXISTS prevent_update_if_policy_deleted();
DROP TRIGGER IF EXISTS trg_set_policy_inactive_on_delete ON plugin_policies;
DROP FUNCTION IF EXISTS set_policy_inactive_on_delete();
ALTER TABLE plugin_policies DROP COLUMN IF EXISTS deleted;
-- +goose StatementEnd
