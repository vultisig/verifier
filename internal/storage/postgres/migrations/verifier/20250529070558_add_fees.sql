-- +goose Up
-- +goose StatementBegin
CREATE TYPE billing_frequency AS ENUM(
    'daily',
    'weekly',
    'biweekly',
    'monthly'
);

CREATE TYPE fee_type AS ENUM(
    'tx',
    'recurring',
    'once'
);

-- Stores info about charging frequencies
CREATE TABLE IF NOT EXISTS plugin_policy_billing(
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    type fee_type NOT NULL, -- Enum type for fee types
    frequency billing_frequency,
    start_date DATE NOT NULL DEFAULT CURRENT_DATE, -- Start date of the billing cycle
    amount int,
    plugin_policy_id uuid NOT NULL, -- Foreign key to plugin_policies
    CONSTRAINT fk_plugin_policy FOREIGN KEY (plugin_policy_id) REFERENCES plugin_policies(id) ON DELETE CASCADE,
    CONSTRAINT frequency_check CHECK (
        (type = 'recurring' AND frequency IS NOT NULL) OR
        (type IN ('tx', 'once') AND frequency IS NULL)
    )
);

CREATE TABLE IF NOT EXISTS fees(
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    plugin_policy_billing_id uuid NOT NULL, -- used for recurring fees
    transaction_id uuid, -- used for tx based fees only
    amount bigint NOT NULL,
    created_at timestamp NOT NULL DEFAULT now(),
    charged_at date NOT NULL DEFAULT now(),
    collected_at timestamp,
    CONSTRAINT fk_billing FOREIGN KEY (plugin_policy_billing_id) REFERENCES plugin_policy_billing(id) ON DELETE CASCADE
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
    from plugin_policy_billing ppb 
    join plugin_policies pp on ppb.plugin_policy_id = pp.id
    LEFT JOIN fees f ON f.plugin_policy_billing_id = ppb.id
    WHERE ppb."type" = 'recurring'
    GROUP BY (ppb.id, pp.id);

CREATE VIEW fees_view AS 
    SELECT pp.id AS policy_id, ppb.id AS billing_id, ppb."type", f.* 
    FROM plugin_policies pp 
    JOIN plugin_policy_billing ppb ON ppb.plugin_policy_id = pp.id 
    JOIN fees f ON f.plugin_policy_billing_id = ppb.id;;

CREATE INDEX idx_fees_plugin_policy_billing_id ON fees(plugin_policy_billing_id);
CREATE INDEX idx_plugin_policy_billing_id ON plugin_policy_billing(id);
CREATE INDEX idx_fees_transaction_id ON fees(transaction_id) WHERE transaction_id IS NOT NULL;
CREATE INDEX idx_fees_billing_date ON fees(charged_at);

-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS fees;
DROP TABLE IF EXISTS plugin_policy_billing;
DROP VIEW IF EXISTS billing_periods;
DROP VIEW IF EXISTS fees_view;
DROP TYPE IF EXISTS billing_frequency;
DROP TYPE IF EXISTS fee_type;
DROP INDEX IF EXISTS idx_fees_plugin_policy_billing_id;
DROP INDEX IF EXISTS idx_fees_transaction_id;
DROP INDEX IF EXISTS idx_fees_billing_date;

-- +goose StatementEnd


