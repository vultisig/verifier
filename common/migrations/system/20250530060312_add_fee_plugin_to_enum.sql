-- +goose Up
-- +goose StatementBegin
ALTER TYPE plugin_id ADD VALUE 'vultisig-fees-feee';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ROLLBACK;
-- +goose StatementEnd
