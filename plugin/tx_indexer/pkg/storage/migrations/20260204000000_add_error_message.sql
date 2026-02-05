-- +goose Up
ALTER TABLE tx_indexer ADD COLUMN error_message TEXT;

-- +goose Down
ALTER TABLE tx_indexer DROP COLUMN error_message;
