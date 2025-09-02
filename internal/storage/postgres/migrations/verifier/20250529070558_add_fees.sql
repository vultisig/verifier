-- +goose Up
-- +goose StatementBegin

CREATE TYPE billing_asset AS ENUM ('usdc');
CREATE TYPE fee_debit_type as ENUM ('fee', 'failed_tx');
CREATE TYPE fee_credit_type as ENUM ('fee_transacted');
CREATE TYPE fee_type as ENUM ('debit', 'credit');

-- Stores info about charging frequencies
CREATE TABLE IF NOT EXISTS plugin_policy_billing(
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    type pricing_type NOT NULL, -- Enum type for fee types
    frequency pricing_frequency,
    start_date DATE NOT NULL DEFAULT CURRENT_DATE, -- Start date of the billing cycle
    amount BIGINT NOT NULL,
    asset pricing_asset NOT NULL,
    plugin_policy_id uuid NOT NULL, -- Foreign key to plugin_policies
    CONSTRAINT fk_plugin_policy FOREIGN KEY (plugin_policy_id) REFERENCES plugin_policies(id) ON DELETE CASCADE,
    CONSTRAINT frequency_check CHECK (
        (type = 'recurring' AND frequency IS NOT NULL) OR
        (type IN ('per-tx', 'once') AND frequency IS NULL)
    )
);

--base table for all fees (append-only)
CREATE TABLE IF NOT EXISTS fees(
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    type fee_type NOT NULL,
    amount BIGINT NOT NULL,
    public_key VARCHAR(66) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    CONSTRAINT fee_positive_amount CHECK (amount > 0),
    ref TEXT --used to store external or internal references, comma separated list of format: "type:id"
);

-- Function to enforce append-only behavior on fees table
CREATE OR REPLACE FUNCTION enforce_fees_append_only()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'UPDATE' THEN
        RAISE EXCEPTION 'UPDATE operations are not allowed on fees table (append-only)';
    END IF;
    
    IF TG_OP = 'DELETE' THEN
        RAISE EXCEPTION 'DELETE operations are not allowed on fees table (append-only)';
    END IF;
    
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

-- Trigger to enforce append-only behavior on fees table
CREATE TRIGGER fees_append_only_trigger
    BEFORE UPDATE OR DELETE ON fees
    FOR EACH ROW
    EXECUTE FUNCTION enforce_fees_append_only();

-- Function to validate that fee public_key matches the plugin_policy public_key
-- Only validates when billing_id is not null (for fee_debits table)
CREATE OR REPLACE FUNCTION validate_fee_public_key(fee_public_key VARCHAR(66), billing_id uuid)
RETURNS BOOLEAN AS $$
BEGIN
    -- If billing_id is NULL, validation passes (for other inherited tables)
    IF billing_id IS NULL THEN
        RETURN TRUE;
    END IF;
    
    -- Otherwise, check that public keys match
    RETURN EXISTS (
        SELECT 1 
        FROM plugin_policy_billing ppb
        JOIN plugin_policies pp ON ppb.plugin_policy_id = pp.id
        WHERE ppb.id = billing_id AND pp.public_key = fee_public_key
    );
END;
$$ LANGUAGE plpgsql STABLE;

CREATE TABLE IF NOT EXISTS fee_debits(
    subtype fee_debit_type NOT NULL,
    plugin_policy_billing_id uuid,
    charged_at DATE NOT NULL DEFAULT now(),
    CONSTRAINT fee_debits_pkey PRIMARY KEY (id),
    CONSTRAINT fk_billing FOREIGN KEY (plugin_policy_billing_id) REFERENCES plugin_policy_billing(id) ON DELETE CASCADE,
    CONSTRAINT fee_debits_billing_id_required CHECK (subtype != 'fee' OR plugin_policy_billing_id IS NOT NULL),
    CONSTRAINT fee_debits_public_key_match CHECK (subtype != 'fee' OR validate_fee_public_key(public_key, plugin_policy_billing_id)),
    CONSTRAINT fee_debits_type_check CHECK (type = 'debit'),
    type fee_type NOT NULL DEFAULT 'debit'
) INHERITS (fees);

CREATE TABLE IF NOT EXISTS fee_credits(
    subtype fee_credit_type NOT NULL,
    CONSTRAINT fee_credits_pkey PRIMARY KEY (id),
    CONSTRAINT fee_credits_type_check CHECK (type = 'credit'),
    type fee_type NOT NULL DEFAULT 'credit'
) INHERITS (fees);

CREATE TYPE fee_batch_status as ENUM ('draft', 'sent', 'completed', 'failed');

CREATE TABLE IF NOT EXISTS fee_batch(
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    public_key VARCHAR(66) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT now(),
    tx_hash VARCHAR(66),
    status fee_batch_status NOT NULL DEFAULT 'draft',
    CONSTRAINT fee_batch_tx_hash_unique UNIQUE (tx_hash),
    CONSTRAINT fee_batch_tx_hash_not_null CHECK (status = 'draft' OR tx_hash IS NOT NULL)
);

-- Function to check if a fee exists in the inheritance hierarchy
CREATE OR REPLACE FUNCTION fee_exists(fee_uuid uuid)
RETURNS BOOLEAN AS $$
BEGIN
    -- Check if the fee exists in the parent table or any inherited tables
    RETURN EXISTS (
        SELECT 1 FROM ONLY fees WHERE id = fee_uuid
        UNION ALL
        SELECT 1 FROM fee_debits WHERE id = fee_uuid
        UNION ALL
        SELECT 1 FROM fee_credits WHERE id = fee_uuid
    );
END;
$$ LANGUAGE plpgsql STABLE;

-- Function to check if a fee's public key matches the batch's public key
CREATE OR REPLACE FUNCTION fee_batch_public_key_match(batch_id uuid, fee_uuid uuid)
RETURNS BOOLEAN AS $$
BEGIN
    -- Check if the fee's public key matches the batch's public key
    RETURN EXISTS (
        SELECT 1 
        FROM fee_batch fb
        WHERE fb.id = batch_id
        AND fb.public_key = (
            SELECT public_key FROM fees WHERE id = fee_uuid
            UNION ALL
            SELECT public_key FROM fee_debits WHERE id = fee_uuid
            UNION ALL
            SELECT public_key FROM fee_credits WHERE id = fee_uuid
            LIMIT 1
        )
    );
END;
$$ LANGUAGE plpgsql STABLE;

CREATE TABLE IF NOT EXISTS fee_batch_members(
    fee_batch_id uuid NOT NULL,
    fee_id uuid NOT NULL,
    CONSTRAINT fee_batch_members_pkey PRIMARY KEY (fee_batch_id, fee_id),
    CONSTRAINT fee_batch_members_fee_id_unique UNIQUE (fee_id),
    CONSTRAINT fk_fee_batch FOREIGN KEY (fee_batch_id) REFERENCES fee_batch(id) ON DELETE CASCADE,
    CONSTRAINT fk_fee CHECK (fee_exists(fee_id)),
    CONSTRAINT fee_batch_public_key_match CHECK (fee_batch_public_key_match(fee_batch_id, fee_id))
);



CREATE VIEW billing_periods AS 
    SELECT pp.id as plugin_policy_id,
    pp.active as active,
    ppb.id as billing_id, 
    ppb.frequency, 
    ppb.amount, 
    COUNT(f.id) as accrual_count,
    COALESCE(SUM(f.amount),0) as total_billed,
    COALESCE(MAX(f.charged_at), ppb.start_date) AS last_billed_date, 
    COALESCE(MAX(f.charged_at), ppb.start_date) +
    CASE ppb.frequency
        WHEN 'daily' THEN INTERVAL '1 day'
        WHEN 'weekly' THEN INTERVAL '1 week'
        WHEN 'biweekly' THEN interval '2 weeks'
        WHEN 'monthly' THEN INTERVAL '1 month'
    END AS next_billing_date
    FROM plugin_policy_billing ppb 
    JOIN plugin_policies pp on ppb.plugin_policy_id = pp.id
    LEFT JOIN fee_debits f ON f.plugin_policy_billing_id = ppb.id
    WHERE ppb."type" = 'recurring'
    GROUP BY (ppb.id, pp.id);

CREATE VIEW fee_debits_view AS 
    SELECT pp.id AS policy_id, pp.plugin_id AS plugin_id, ppb.id AS billing_id, f.public_key, ppb."type", 
           f.id, f.amount, f.created_at, f.type AS fee_type, f.plugin_policy_billing_id, f.charged_at
    FROM plugin_policies pp 
    JOIN plugin_policy_billing ppb ON ppb.plugin_policy_id = pp.id 
    JOIN fee_debits f ON f.plugin_policy_billing_id = ppb.id;

CREATE VIEW fees_joined AS
    SELECT id, public_key, "type", "subtype"::text as "subtype", created_at, amount FROM fee_credits fc 
    UNION ALL
    SELECT id, public_key, "type", "subtype"::text as "subtype", created_at, amount FROM fee_debits fd;

CREATE VIEW fee_balance AS
    SELECT public_key, SUM(
    CASE 
        WHEN type = 'credit' THEN -amount
        ELSE amount
    END
    ) AS total_owed, 
    COUNT(*) FILTER (WHERE type = 'debit') AS total_debits,
    COUNT(*) FILTER (WHERE type = 'credit') AS total_credits
    FROM fees_joined GROUP BY public_key ;

-- Function to get fee balance for specific fee IDs
CREATE OR REPLACE FUNCTION fee_balance_for_ids(fee_ids uuid[])
RETURNS TABLE(public_key VARCHAR(66), total_owed BIGINT, total_debits BIGINT, total_credits BIGINT) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        fj.public_key, 
        SUM(
            CASE 
                WHEN fj.type = 'credit' THEN -fj.amount
                ELSE fj.amount
            END
        )::BIGINT AS total_owed,
        COUNT(*) FILTER (WHERE fj.type = 'debit')::BIGINT AS total_debits,
        COUNT(*) FILTER (WHERE fj.type = 'credit')::BIGINT AS total_credits
    FROM fees_joined fj 
    WHERE fj.id = ANY(fee_ids)
    GROUP BY fj.public_key;
END;
$$ LANGUAGE plpgsql STABLE;


CREATE INDEX idx_plugin_policy_billing_id ON plugin_policy_billing(id);

CREATE INDEX idx_fees_created_at ON fees(created_at);

CREATE INDEX idx_fee_debits_plugin_policy_billing_id ON fee_debits(plugin_policy_billing_id);
CREATE INDEX idx_fee_debits_billing_date ON fee_debits(charged_at);

CREATE INDEX idx_fee_batch_transaction_hash ON fee_batch(tx_hash) WHERE tx_hash IS NOT NULL;

-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DROP VIEW IF EXISTS fee_balance;
DROP VIEW IF EXISTS fees_joined;
DROP VIEW IF EXISTS fee_debits_view;
DROP TRIGGER IF EXISTS fees_append_only_trigger ON fees;
DROP TABLE IF EXISTS fee_credits;
DROP TABLE IF EXISTS fee_debits;
DROP TABLE IF EXISTS fees;
DROP TABLE IF EXISTS plugin_policy_billing;
DROP VIEW IF EXISTS billing_periods;
DROP FUNCTION IF EXISTS enforce_fees_append_only();
DROP FUNCTION IF EXISTS validate_fee_public_key(VARCHAR(66), uuid);
DROP TYPE IF EXISTS fee_debit_type;
DROP TYPE IF EXISTS fee_credit_type;
DROP TYPE IF EXISTS billing_asset;
DROP INDEX IF EXISTS idx_plugin_policy_billing_id;
DROP INDEX IF EXISTS idx_fees_created_at;
DROP INDEX IF EXISTS idx_fee_debits_plugin_policy_billing_id;
DROP INDEX IF EXISTS idx_fee_debits_billing_date;
DROP INDEX IF EXISTS idx_fee_batch_transaction_hash;

-- +goose StatementEnd


