-- +goose Up
-- +goose StatementBegin

-- Add unique constraint to ensure only one policy per public key for vultisig-fees-feee plugin
DROP INDEX IF EXISTS unique_fees_policy_per_public_key;
CREATE UNIQUE INDEX unique_fees_policy_per_public_key
ON plugin_policies (plugin_id, public_key)
WHERE plugin_id = 'vultisig-fees-feee' AND active = true;

-- Create a function to check if there are active fees for a public key
CREATE OR REPLACE FUNCTION check_active_fees_for_public_key()
RETURNS TRIGGER AS $$
BEGIN
    -- Lowered lock level to SHARE MODE to reduce deadlock risk with concurrent INSERTs.
    -- This should be sufficient to prevent concurrent modifications for our check.
    LOCK TABLE fees IN SHARE MODE;
    -- If we're deleting a vultisig-fees-feee policy
    IF OLD.plugin_id = 'vultisig-fees-feee' THEN
        -- Check if there are any active fees for this public key
        IF EXISTS (
            SELECT 1 
            FROM fees_view fv 
            WHERE fv.public_key = OLD.public_key 
            AND fv.policy_id = OLD.id
        ) THEN
            RAISE EXCEPTION 'Cannot delete plugin policy: active fees exist for public key %', OLD.public_key;
        END IF;
    END IF;
    
    RETURN OLD;
END;
$$ LANGUAGE plpgsql;

-- Create trigger to prevent deletion when active fees exist
CREATE TRIGGER prevent_fees_policy_deletion_with_active_fees
    BEFORE DELETE ON plugin_policies
    FOR EACH ROW
    EXECUTE FUNCTION check_active_fees_for_public_key();

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Drop the trigger
DROP TRIGGER IF EXISTS prevent_fees_policy_deletion_with_active_fees ON plugin_policies;

-- Drop the function
DROP FUNCTION IF EXISTS check_active_fees_for_public_key();

-- Drop the unique partial index
DROP INDEX IF EXISTS unique_fees_policy_per_public_key;

-- +goose StatementEnd 