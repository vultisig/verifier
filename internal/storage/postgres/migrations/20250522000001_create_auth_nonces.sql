-- +goose Up
-- +goose StatementBegin
CREATE TABLE auth_nonces (
    nonce VARCHAR(32) PRIMARY KEY,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    message_expiry TIMESTAMP NOT NULL,
    public_key VARCHAR(66) NOT NULL
);

-- Create indexes for efficient cleanup and lookups
CREATE INDEX idx_auth_nonces_created_at ON auth_nonces(created_at);
CREATE INDEX idx_auth_nonces_message_expiry ON auth_nonces(message_expiry);
-- +goose StatementEnd 

-- +goose Down 
-- +goose StatementBegin
DROP TABLE IF EXISTS auth_nonces;
-- +goose StatementEnd 