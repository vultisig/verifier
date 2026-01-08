-- +goose Up
ALTER TABLE fees ADD COLUMN plugin_id VARCHAR(255);

-- Backfill plugin_id from policy_id where possible
UPDATE fees f
SET plugin_id = pp.plugin_id
FROM plugin_policies pp
WHERE f.policy_id = pp.id AND f.plugin_id IS NULL;

-- Backfill plugin_id from underlying_id for installation fees
UPDATE fees
SET plugin_id = underlying_id
WHERE underlying_type = 'plugin' AND fee_type = 'installation_fee' AND plugin_id IS NULL;

CREATE INDEX idx_fees_plugin_id ON fees(plugin_id) WHERE plugin_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_fees_plugin_id;
ALTER TABLE fees DROP COLUMN plugin_id;
