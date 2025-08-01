-- +goose Up
-- +goose StatementBegin
-- Create enum type for fee_account.type
CREATE TYPE treasury_ledger_entry_type AS ENUM (
    'fee_credit',
    'developer_payout',
    'refund',
    'credit_adjustment',
    'debit_adjustment'
);

-- append only table for fee accounts
CREATE TABLE IF NOT EXISTS treasury_ledger (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    amount BIGINT NOT NULL, -- positive for credits, negative for debits
    type treasury_ledger_entry_type NOT NULL,

    -- Flexible references
    fee_id UUID, -- the id of the fee that was collected
    developer_id UUID, -- the id of the developer credited or debited
    tx_hash VARCHAR(66), -- the hash of the transaction that sent a payment
    reference VARCHAR(255), -- for any external references

    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    -- fee id can't be null if type is fee_credit
    CONSTRAINT chk_fee_credit CHECK (
        (type != 'fee_credit') OR fee_id IS NOT NULL
    ),

    -- developer id and tx id can't be null if type is developer_payout
    CONSTRAINT chk_developer_payout CHECK (
        (type != 'developer_payout') OR (developer_id IS NOT NULL AND tx_hash IS NOT NULL)
    ),

    -- tx id can't be null if type is refund
    CONSTRAINT chk_refund CHECK (
        (type != 'refund') OR tx_hash IS NOT NULL
    )
);

CREATE INDEX idx_ledger_fee_id ON treasury_ledger(fee_id);
CREATE INDEX idx_ledger_developer_id ON treasury_ledger(developer_id);
CREATE INDEX idx_ledger_tx_hash ON treasury_ledger(tx_hash);

-- Make the table append-only
CREATE OR REPLACE FUNCTION ledger_no_update_delete()
    RETURNS trigger AS $$
    BEGIN
        RAISE EXCEPTION 'treasury_ledger is append-only: updates and deletes are not allowed';
    END;
    $$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS ledger_no_update ON treasury_ledger;
    CREATE TRIGGER ledger_no_update
    BEFORE UPDATE OR DELETE ON treasury_ledger
    FOR EACH ROW EXECUTE FUNCTION ledger_no_update_delete();

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin

DROP TRIGGER IF EXISTS ledger_no_update ON treasury_ledger;
DROP FUNCTION IF EXISTS ledger_no_update_delete();
DROP INDEX IF EXISTS idx_ledger_fee_id;
DROP INDEX IF EXISTS idx_ledger_developer_id;
DROP INDEX IF EXISTS idx_ledger_tx_hash;
DROP TABLE IF EXISTS treasury_ledger;
DROP TYPE IF EXISTS treasury_ledger_entry_type;
-- +goose StatementEnd
