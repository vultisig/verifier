-- +goose Up
-- +goose StatementBegin
BEGIN
CREATE TABLE tx_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    plugin_id UUID NOT NULL,
    tx_hash TINYTEXT,
    chain TINYTEXT NOT NULL,
    policy_id UUID,
    from_public_key TINYTEXT NOT NULL,
    proposed_tx_object JSONB NOT NULL,

    -- PROPOSED -> VERIFIED -> SIGNED
    status TINYTEXT NOT NULL DEFAULT 'PROPOSED',

    -- null -> PENDING -> SUCCESS / FAIL
    status_onchain TINYTEXT,

    -- is Tx stuck in PENDING_ONCHAIN status more than configured timeout,
    -- flag to exclude it from on-chain status polling
    lost BOOLEAN NOT NULL DEFAULT false,

    broadcasted_at TIMESTAMP,
    -- when signed tx broadcasted on-chain

    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
)

CREATE INDEX idx_tx_history_status_lost ON tx_history(status, lost)
END
-- +goose StatementEnd
