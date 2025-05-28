-- +goose Up
-- +goose StatementBegin
CREATE TYPE plugin_id AS ENUM (
    'vultisig-dca-0000',
    'vultisig-payroll-0000'
);

CREATE TABLE plugin_policies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    public_key TEXT NOT NULL,
    plugin_id plugin_id NOT NULL,
    plugin_version TEXT NOT NULL,
    policy_version INTEGER NOT NULL,
    signature TEXT NOT NULL,
    recipe TEXT NOT NULL,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_plugin_policies_plugin_id ON plugin_policies(plugin_id);
CREATE INDEX idx_plugin_policies_public_key ON plugin_policies(public_key);
CREATE INDEX idx_plugin_policies_active ON plugin_policies(active);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ROLLBACK;
-- +goose StatementEnd