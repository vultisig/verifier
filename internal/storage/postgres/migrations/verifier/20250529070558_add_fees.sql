-- +goose Up
-- +goose StatementBegin
CREATE TYPE billing_frequency AS ENUM(
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
    start_date int NOT NULL DEFAULT 1,
    plugin_policy_id uuid NOT NULL, -- Foreign key to plugin_policies
    CONSTRAINT fk_plugin_policy FOREIGN KEY (plugin_policy_id) REFERENCES plugin_policies(id) ON DELETE CASCADE,
    CONSTRAINT only_first_of_month CHECK (start_date = 1)
);

CREATE TABLE IF NOT EXISTS fees(
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    type fee_type NOT NULL, -- Enum type for fee types
    plugin_policy_billing_id uuid NOT NULL, -- used for recurring fees
    transaction_id uuid, -- used for tx based fees only
    billing_date date NOT NULL, -- used for tx based fees only
    amount bigint NOT NULL,
    created_at timestamp DEFAULT now(),
    collected_at timestamp,
    CONSTRAINT fk_billing FOREIGN KEY (plugin_policy_billing_id) REFERENCES plugin_policy_billing(id) ON DELETE CASCADE
);

-- +goose StatementEnd
-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS fees;

DROP TABLE IF EXISTS plugin_policy_billing;

DROP TYPE IF EXISTS billing_frequency;

DROP TYPE IF EXISTS fee_type;

-- +goose StatementEnd
