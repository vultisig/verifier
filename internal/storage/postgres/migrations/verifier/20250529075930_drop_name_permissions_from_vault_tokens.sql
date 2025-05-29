-- +goose Up
-- +goose StatementBegin
ALTER TABLE vault_tokens DROP COLUMN name;
ALTER TABLE vault_tokens DROP COLUMN permissions;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ROLLBACK;
-- +goose StatementEnd
