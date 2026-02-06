-- +goose Up
ALTER TABLE tx_indexer ADD COLUMN error_message TEXT;

-- +goose Down
