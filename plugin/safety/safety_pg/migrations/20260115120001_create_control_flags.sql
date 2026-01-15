-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS control_flags (
    key         TEXT PRIMARY KEY,
    enabled     BOOLEAN NOT NULL,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS control_flags;
-- +goose StatementEnd
