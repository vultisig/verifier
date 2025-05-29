-- +goose Up
-- +goose StatementBegin
ALTER TABLE vault_tokens ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ROLLBACK;
-- +goose StatementEnd
