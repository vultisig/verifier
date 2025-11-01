-- +goose Up
-- +goose StatementBegin
DROP TRIGGER IF EXISTS trg_prevent_tx_indexer_update_if_policy_deleted ON tx_indexer;
DROP FUNCTION IF EXISTS prevent_tx_indexer_update_if_policy_deleted();
-- +goose StatementEnd
    
-- +goose Down
-- +goose StatementBegin
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
