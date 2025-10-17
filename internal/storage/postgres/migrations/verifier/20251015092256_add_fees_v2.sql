-- +goose Up

-- =====================================================
-- ENUM TYPES
-- =====================================================

CREATE TYPE transaction_type AS ENUM ('debit', 'credit');

CREATE TYPE batch_status AS ENUM ('SIGNED', 'BATCHED', 'FAILED', 'COMPLETED');

-- =====================================================
-- TABLES
-- =====================================================

CREATE TABLE fees
(
    id               BIGSERIAL PRIMARY KEY,
    policy_id        uuid,
    public_key       TEXT             NOT NULL,
    transaction_type transaction_type NOT NULL,
    amount           bigint           NOT NULL CHECK (amount > 0),
    created_at       TIMESTAMPTZ      NOT NULL DEFAULT NOW(),
    fee_type         TEXT             NOT NULL,
    metadata         JSONB,
    underlying_type  TEXT             NOT NULL,
    underlying_id    TEXT             NOT NULL,

    CONSTRAINT unique_fee_per_entity UNIQUE (fee_type, underlying_type, underlying_id),
    CONSTRAINT policy_id_required_for_policies CHECK (
        (underlying_type = 'policy' AND policy_id IS NOT NULL) OR
        (underlying_type != 'policy')
        )
);

CREATE TABLE fee_batches
(
    id               BIGSERIAL PRIMARY KEY,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    total_value      bigint       NOT NULL CHECK (total_value >= 0),
    status           batch_status NOT NULL DEFAULT 'BATCHED',
    batch_cutoff     INTEGER      NOT NULL,
    collection_tx_id TEXT
);

CREATE TABLE fee_batch_members
(
    batch_id BIGINT NOT NULL REFERENCES fee_batches (id) ON DELETE CASCADE,
    fee_id   BIGINT NOT NULL UNIQUE REFERENCES fees (id) ON DELETE RESTRICT,
    PRIMARY KEY (batch_id, fee_id)
);

-- =====================================================
-- TRIGGER FUNCTIONS
-- =====================================================

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION prevent_fee_deletion()
    RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'DELETE operation not allowed on fees table. Fee records are immutable for audit compliance.'
        USING HINT = 'create a compensating transaction (credit fee) instead.';
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

-- =====================================================
-- TRIGGERS
-- =====================================================

CREATE TRIGGER trigger_prevent_fee_deletion
    BEFORE DELETE
    ON fees
    FOR EACH ROW
    EXECUTE FUNCTION prevent_fee_deletion();

-- =====================================================
-- INDEXES
-- =====================================================

CREATE INDEX idx_fees_public_key ON fees (public_key);

CREATE INDEX idx_fees_policy_id ON fees (policy_id) WHERE policy_id IS NOT NULL;

CREATE INDEX idx_fees_created_at ON fees (created_at DESC);

CREATE INDEX idx_fees_underlying_entity ON fees (underlying_type, underlying_id);

CREATE INDEX idx_fees_transaction_type ON fees (transaction_type);

CREATE INDEX idx_fees_metadata_gin ON fees USING GIN(metadata) WHERE metadata IS NOT NULL;

CREATE INDEX idx_fee_batches_status ON fee_batches (status);

CREATE INDEX idx_fee_batches_created_at ON fee_batches (created_at DESC);

CREATE INDEX idx_fee_batches_collection_tx_id ON fee_batches (collection_tx_id) WHERE collection_tx_id IS NOT NULL;

CREATE UNIQUE INDEX idx_unique_installation_fee_per_plugin_user
    ON fees(underlying_id, public_key)
    WHERE fee_type = 'installation_fee' AND underlying_type = 'plugin';

-- +goose Down
-- Drop tables in reverse order to respect foreign key constraints
DROP TABLE IF EXISTS fee_batch_members;
DROP TABLE IF EXISTS fees;
DROP TABLE IF EXISTS fee_batches;

-- Drop enum types
DROP TYPE IF EXISTS transaction_type;
DROP TYPE IF EXISTS batch_status;
