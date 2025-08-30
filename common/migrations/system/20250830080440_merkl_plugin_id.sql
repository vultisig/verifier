-- +goose Up
-- +goose StatementBegin
ALTER TYPE plugin_id ADD VALUE 'nbits-labs-merkle-e93d';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ROLLBACK;
-- +goose StatementEnd
