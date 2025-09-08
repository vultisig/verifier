-- +goose Up
-- +goose StatementBegin
ALTER TYPE plugin_id ADD VALUE 'vultisig-tester-ae1d';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ROLLBACK;
-- +goose StatementEnd
