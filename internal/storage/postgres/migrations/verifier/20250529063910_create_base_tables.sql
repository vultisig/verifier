-- +goose Up
-- +goose StatementBegin
-- Create category enum
CREATE TYPE plugin_category AS ENUM ('ai-agent', 'plugin');
CREATE TYPE pricing_frequency AS ENUM ('daily','weekly', 'biweekly', 'monthly');
CREATE TYPE pricing_metric AS ENUM ('fixed');
CREATE TYPE pricing_type AS ENUM ('once', 'recurring', 'per-tx');
CREATE TYPE pricing_asset AS ENUM ('usdc');

-- Plugins table (simplified)
CREATE TABLE plugins (
    id plugin_id PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    server_endpoint TEXT NOT NULL,
    category plugin_category NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Pricings table
CREATE TABLE pricings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type pricing_type NOT NULL,
    frequency pricing_frequency DEFAULT NULL,
    amount BIGINT NOT NULL,
    asset pricing_asset NOT NULL,
    metric pricing_metric NOT NULL,
    plugin_id plugin_id NOT NULL REFERENCES plugins(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
    CONSTRAINT frequency_check CHECK (
        (type = 'recurring' AND frequency IS NOT NULL) OR
        (type IN ('per-tx', 'once') AND frequency IS NULL)
    )
);

-- Plugin policy sync
CREATE TABLE plugin_policy_sync (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    policy_id UUID NOT NULL REFERENCES plugin_policies(id) ON DELETE CASCADE,
    plugin_id plugin_id NOT NULL REFERENCES plugins(id) ON DELETE CASCADE,
    sync_type INT NOT NULL, -- 0: add policy, 1: update, 2: delete
    signature TEXT,
    status INT NOT NULL DEFAULT 0, -- 0: not sync, 1: synced, 2: failed
    reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Vault tokens
CREATE TABLE vault_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    token_id VARCHAR(255) NOT NULL UNIQUE,
    public_key VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    permissions JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL,
    last_used_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ
);

-- Tags
CREATE TABLE tags (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Plugin tags junction table
CREATE TABLE plugin_tags (
    plugin_id plugin_id REFERENCES plugins(id) ON DELETE CASCADE,
    tag_id UUID REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (plugin_id, tag_id)
);

-- Reviews
CREATE TABLE reviews (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    plugin_id plugin_id REFERENCES plugins(id) ON DELETE CASCADE,
    public_key TEXT NOT NULL,
    rating INTEGER NOT NULL CHECK (rating >= 1 AND rating <= 5),
    comment TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Plugin ratings (aggregated)
CREATE TABLE plugin_ratings (
    plugin_id plugin_id PRIMARY KEY REFERENCES plugins(id) ON DELETE CASCADE,
    avg_rating DECIMAL(3, 2) NOT NULL DEFAULT 0,
    total_ratings INTEGER NOT NULL DEFAULT 0,
    rating_1_count INTEGER NOT NULL DEFAULT 0,
    rating_2_count INTEGER NOT NULL DEFAULT 0,
    rating_3_count INTEGER NOT NULL DEFAULT 0,
    rating_4_count INTEGER NOT NULL DEFAULT 0,
    rating_5_count INTEGER NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_plugin_policy_sync_policy_id ON plugin_policy_sync(policy_id);
CREATE INDEX idx_vault_tokens_public_key ON vault_tokens(public_key);
CREATE INDEX idx_vault_tokens_token_id ON vault_tokens(token_id);
CREATE INDEX idx_reviews_plugin_id ON reviews(plugin_id);
CREATE INDEX idx_reviews_public_key ON reviews(public_key);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ROLLBACK;
-- +goose StatementEnd
