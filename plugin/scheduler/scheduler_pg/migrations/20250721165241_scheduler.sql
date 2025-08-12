-- +goose Up
-- +goose StatementBegin
BEGIN;
CREATE TABLE scheduler (
    policy_id UUID PRIMARY KEY,
    next_execution TIMESTAMP NOT NULL
);

CREATE INDEX idx_scheduler_next_execution ON scheduler(next_execution);
END;
-- +goose StatementEnd
