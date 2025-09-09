-- +goose Up
-- +goose StatementBegin

CREATE TYPE treasury_ledger_type AS ENUM (
    'debit',
    'credit'
);

CREATE TYPE treasury_ledger_entry_type AS ENUM (
    'fee_batch_collection', -- subtype for fee batch collections (debit)
    'developer_payout', -- subtype for developer payouts (credit)
    'failed_tx', -- subtype for failed transactions (debit)
    'vultisig_dues' -- subtype for vultisig dues (debit)
);

CREATE TYPE treasury_batch_status AS ENUM (
    'draft',
    'sent',
    'completed',
    'failed'
);

-- append only table accounts
CREATE TABLE IF NOT EXISTS treasury_ledger (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    developer_id UUID NOT NULL,
    amount BIGINT NOT NULL, -- always positive
    type treasury_ledger_type NOT NULL, -- 'debit' or 'credit'
    subtype treasury_ledger_entry_type NOT NULL, -- specific operation type
    ref TEXT, -- for any external references
    created_at TIMESTAMP NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS treasury_batch (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    tx_hash VARCHAR(66),
    developer_id UUID NOT NULL,
    status treasury_batch_status NOT NULL DEFAULT 'draft',
    CONSTRAINT treasury_batch_tx_hash_unique UNIQUE (tx_hash),
    CONSTRAINT treasury_batch_tx_hash_not_null CHECK (status = 'draft' OR tx_hash IS NOT NULL)
);

CREATE TABLE IF NOT EXISTS treasury_batch_members (
    batch_id UUID NOT NULL,
    treasury_ledger_record_id UUID NOT NULL UNIQUE,
    CONSTRAINT fk_batch FOREIGN KEY (batch_id) REFERENCES treasury_batch(id) ON DELETE CASCADE,
    CONSTRAINT fk_treasury_ledger_record FOREIGN KEY (treasury_ledger_record_id) REFERENCES treasury_ledger(id) ON DELETE CASCADE
);

CREATE VIEW treasury_batch_members_view AS
    SELECT batch_id, COUNT(tbm.treasury_ledger_record_id) as record_count, SUM(
        CASE
            WHEN tl.type = 'credit' THEN -tl.amount
            ELSE tl.amount
        END
    ) as total_amount
    FROM treasury_batch_members tbm
    JOIN treasury_ledger tl ON tbm.treasury_ledger_record_id = tl.id
    GROUP BY batch_id;

-- entries that correspond to a fee batch collection go here (debits that are owed to a developer)
CREATE TABLE IF NOT EXISTS treasury_ledger_fee_batch_collection (
  CONSTRAINT treasury_ledger_fee_batch_collection_pkey PRIMARY KEY (id),
  fee_batch_id UUID NOT NULL,
  type treasury_ledger_type NOT NULL DEFAULT 'debit',
  subtype treasury_ledger_entry_type NOT NULL DEFAULT 'fee_batch_collection',
  CONSTRAINT treasury_must_be_debit CHECK (type = 'debit'),
  CONSTRAINT treasury_must_be_fee_batch_collection CHECK (subtype = 'fee_batch_collection')
) INHERITS (treasury_ledger);

CREATE TABLE IF NOT EXISTS treasury_ledger_developer_payout (
  CONSTRAINT treasury_ledger_developer_payout_pkey PRIMARY KEY (id),
  treasury_batch_id UUID NOT NULL,
  type treasury_ledger_type NOT NULL DEFAULT 'credit',
  subtype treasury_ledger_entry_type NOT NULL DEFAULT 'developer_payout',
  CONSTRAINT treasury_must_be_credit CHECK (type = 'credit'),
  CONSTRAINT treasury_must_be_developer_payout CHECK (subtype = 'developer_payout')
) INHERITS (treasury_ledger);

CREATE TABLE IF NOT EXISTS treasury_ledger_failed_tx (
  CONSTRAINT treasury_ledger_failed_tx_pkey PRIMARY KEY (id),
  treasury_batch_id UUID NOT NULL,
  type treasury_ledger_type NOT NULL DEFAULT 'debit',
  subtype treasury_ledger_entry_type NOT NULL DEFAULT 'failed_tx',
  CONSTRAINT treasury_must_be_debit CHECK (type = 'debit'),
  CONSTRAINT treasury_must_be_failed_tx CHECK (subtype = 'failed_tx')
) INHERITS (treasury_ledger);

CREATE TABLE IF NOT EXISTS treasury_vultisig_debit (
  CONSTRAINT treasury_vultisig_debit_pkey PRIMARY KEY (id),
  fee_batch_id UUID NOT NULL,
  type treasury_ledger_type NOT NULL DEFAULT 'debit',
  subtype treasury_ledger_entry_type NOT NULL DEFAULT 'vultisig_dues',
  CONSTRAINT treasury_must_be_debit CHECK (type = 'debit'),
  CONSTRAINT treasury_must_be_vultisig_dues CHECK (subtype = 'vultisig_dues')
) INHERITS (treasury_ledger);

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_treasury_ledger_type ON treasury_ledger(type);
CREATE INDEX IF NOT EXISTS idx_treasury_ledger_subtype ON treasury_ledger(subtype);
CREATE INDEX IF NOT EXISTS idx_treasury_ledger_created_at ON treasury_ledger(created_at);
CREATE INDEX IF NOT EXISTS idx_treasury_ledger_fee_batch_collection_fee_batch_id ON treasury_ledger_fee_batch_collection(fee_batch_id);
CREATE INDEX IF NOT EXISTS idx_treasury_ledger_developer_payout_treasury_batch_id ON treasury_ledger_developer_payout(treasury_batch_id);
CREATE INDEX IF NOT EXISTS idx_treasury_ledger_failed_tx_treasury_batch_id ON treasury_ledger_failed_tx(treasury_batch_id);
CREATE INDEX IF NOT EXISTS idx_treasury_vultisig_debit_fee_batch_id ON treasury_vultisig_debit(fee_batch_id);

-- Make the treasury_ledger tables append-only
CREATE OR REPLACE FUNCTION treasury_ledger_no_update_delete()
    RETURNS trigger AS $$
    BEGIN
        RAISE EXCEPTION 'treasury_ledger tables are append-only: updates and deletes are not allowed';
    END;
    $$ LANGUAGE plpgsql;

-- Apply append-only triggers to all treasury_ledger tables
-- Note: Triggers on parent tables automatically apply to inherited tables in PostgreSQL
DROP TRIGGER IF EXISTS treasury_ledger_no_update ON treasury_ledger;
CREATE TRIGGER treasury_ledger_no_update
    BEFORE UPDATE OR DELETE ON treasury_ledger
    FOR EACH ROW EXECUTE FUNCTION treasury_ledger_no_update_delete();

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

-- Drop triggers first
DROP TRIGGER IF EXISTS treasury_ledger_no_update ON treasury_ledger;

-- Drop function
DROP FUNCTION IF EXISTS treasury_ledger_no_update_delete();

-- Drop indexes
DROP INDEX IF EXISTS idx_treasury_vultisig_debit_fee_batch_id;
DROP INDEX IF EXISTS idx_treasury_ledger_failed_tx_treasury_batch_id;
DROP INDEX IF EXISTS idx_treasury_ledger_developer_payout_treasury_batch_id;
DROP INDEX IF EXISTS idx_treasury_ledger_fee_batch_collection_fee_batch_id;
DROP INDEX IF EXISTS idx_treasury_ledger_created_at;
DROP INDEX IF EXISTS idx_treasury_ledger_subtype;
DROP INDEX IF EXISTS idx_treasury_ledger_type;

-- Drop inherited tables first (child tables before parent)
DROP TABLE IF EXISTS treasury_vultisig_debit;
DROP TABLE IF EXISTS treasury_ledger_failed_tx;
DROP TABLE IF EXISTS treasury_ledger_developer_payout;
DROP TABLE IF EXISTS treasury_ledger_fee_batch_collection;

-- Drop views
DROP VIEW IF EXISTS treasury_batch_members_view;

-- Drop main tables
DROP TABLE IF EXISTS treasury_batch_members;
DROP TABLE IF EXISTS treasury_batch;
DROP TABLE IF EXISTS treasury_ledger;

-- Drop types (in reverse order of dependency)
DROP TYPE IF EXISTS treasury_batch_status;
DROP TYPE IF EXISTS treasury_ledger_entry_type;
DROP TYPE IF EXISTS treasury_ledger_type;

-- +goose StatementEnd
