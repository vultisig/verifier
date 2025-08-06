-- +goose Up
-- +goose StatementBegin
CREATE TABLE "plugin_policies" (
    id UUID DEFAULT gen_random_uuid() NOT NULL,
    public_key VARCHAR(255) NOT NULL,
    plugin_id VARCHAR(255) NOT NULL,
    plugin_version VARCHAR(255) NOT NULL,
    policy_version INTEGER NOT NULL,
    signature VARCHAR(255) NOT NULL,
    recipe VARCHAR(255) NOT NULL,
    active BOOLEAN DEFAULT true NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT now() NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT now() NOT NULL,
    deleted BOOLEAN DEFAULT false NOT NULL
)
-- +goose StatementEnd
