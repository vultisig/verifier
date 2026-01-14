-- +goose Up
-- +goose StatementBegin
ALTER TABLE tx_indexer ADD COLUMN amount TEXT;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE tx_indexer DROP COLUMN amount;
-- +goose StatementEnd
