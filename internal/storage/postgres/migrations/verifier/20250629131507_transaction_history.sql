-- +goose Up
-- +goose StatementBegin
CREATE TYPE transaction_status AS ENUM (
  'SIGNING_IN_PROGRESS',
  'SIGNING_FAILED',
  'SIGNED',
  'BROADCAST',
  'PENDING',
  'MINED',
  'REJECTED'
);
CREATE TABLE IF NOT EXISTS transaction_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid() NOT NULL,
    policy_id UUID NOT NULL,
    tx_body TEXT NOT NULL,
    tx_hash TEXT NOT NULL,
    status transaction_status NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    metadata JSONB  NOT NULL DEFAULT '{}'::jsonb,
    error_message TEXT,
    CONSTRAINT fk_transaction_history_plugin_policy FOREIGN KEY (policy_id)
        REFERENCES plugin_policies(id) ON DELETE CASCADE
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS transaction_history;
DROP TYPE IF EXISTS transaction_status;
-- +goose StatementEnd
