-- +goose Up
CREATE UNIQUE INDEX IF NOT EXISTS idx_unique_trial_fee
    ON fees(public_key)
    WHERE fee_type = 'trial';

-- +goose Down
DROP INDEX IF EXISTS idx_unique_trial_fee;