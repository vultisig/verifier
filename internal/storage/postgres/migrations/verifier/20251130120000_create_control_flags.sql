-- +goose Up
CREATE TABLE control_flags (
    key         TEXT PRIMARY KEY,
    enabled     BOOLEAN NOT NULL,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE control_flags;