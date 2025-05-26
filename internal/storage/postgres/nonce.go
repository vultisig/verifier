package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/vultisig/verifier/config"
)

// CheckNonceExists checks if a nonce has been used by the same public key and is still valid
func (p *PostgresBackend) CheckNonceExists(ctx context.Context, nonce string, publicKey string) (bool, error) {
	var exists bool
	// Check if nonce exists for this public key and message hasn't expired
	err := p.pool.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM auth_nonces WHERE nonce = $1 AND public_key = $2 AND message_expiry > NOW())",
		nonce, publicKey,
	).Scan(&exists)
	return exists, err
}

// StoreNonce stores a nonce in the database with its message expiry time
func (p *PostgresBackend) StoreNonce(ctx context.Context, nonce string, publicKey string, messageExpiry time.Time, createdAt time.Time) error {
	_, err := p.pool.Exec(ctx,
		"INSERT INTO auth_nonces (nonce, public_key, message_expiry, created_at) VALUES ($1, $2, $3, $4)",
		nonce, publicKey, messageExpiry, createdAt,
	)
	return err
}

// CleanupExpiredNonces removes nonces that are older than 2*EXPIRY_WINDOW
func (p *PostgresBackend) CleanupExpiredNonces(ctx context.Context) error {
	cfg, err := config.ReadVerifierConfig()
	if err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}

	// Delete records older than 2*EXPIRY_WINDOW
	_, err = p.pool.Exec(ctx,
		fmt.Sprintf("DELETE FROM auth_nonces WHERE created_at < NOW() - INTERVAL '%d minutes'",
			2*cfg.Auth.NonceExpiryMinutes),
	)
	return err
}
