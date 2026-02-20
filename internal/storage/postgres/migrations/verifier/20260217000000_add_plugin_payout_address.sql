-- +goose Up
-- +goose StatementBegin
ALTER TABLE plugins ADD COLUMN payout_address TEXT;

CREATE INDEX idx_plugins_payout_address ON plugins(payout_address)
WHERE payout_address IS NOT NULL;
-- +goose StatementEnd