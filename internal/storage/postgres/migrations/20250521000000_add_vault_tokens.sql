-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS vault_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    public_key TEXT NOT NULL,
    token_id TEXT NOT NULL,  -- First part of JWT for identification
    issued_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    is_revoked BOOLEAN DEFAULT FALSE,
    last_used_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,    
    CONSTRAINT unique_token_id UNIQUE (token_id)
);

-- Indexes for faster lookups
CREATE INDEX IF NOT EXISTS idx_vault_tokens_public_key ON vault_tokens(public_key);
CREATE INDEX IF NOT EXISTS idx_vault_tokens_token_id ON vault_tokens(token_id);
CREATE INDEX IF NOT EXISTS idx_vault_tokens_is_revoked ON vault_tokens(is_revoked);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS vault_tokens;
-- +goose StatementEnd 