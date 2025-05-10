package postgres

import (
	"context"
	"time"

	"github.com/vultisig/verifier/internal/types"
)

func (p *PostgresBackend) CreateVaultToken(ctx context.Context, token types.VaultTokenCreate) (*types.VaultToken, error) {
	query := `
		INSERT INTO vault_tokens (public_key, token_id, issued_at, expires_at, is_revoked, last_used_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, public_key, token_id, issued_at, expires_at, is_revoked, last_used_at, created_at, updated_at`

	now := time.Now()
	var vaultToken types.VaultToken
	err := p.pool.QueryRow(ctx, query,
		token.PublicKey,
		token.TokenID,
		now,
		token.ExpiresAt,
		false,
		now,
		now,
		now,
	).Scan(
		&vaultToken.ID,
		&vaultToken.PublicKey,
		&vaultToken.TokenID,
		&vaultToken.IssuedAt,
		&vaultToken.ExpiresAt,
		&vaultToken.IsRevoked,
		&vaultToken.LastUsedAt,
		&vaultToken.CreatedAt,
		&vaultToken.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &vaultToken, nil
}

func (p *PostgresBackend) GetVaultToken(ctx context.Context, tokenID string) (*types.VaultToken, error) {
	query := `
		SELECT id, public_key, token_id, issued_at, expires_at, is_revoked, last_used_at, created_at, updated_at
		FROM vault_tokens
		WHERE token_id = $1`

	var vaultToken types.VaultToken
	err := p.pool.QueryRow(ctx, query, tokenID).Scan(
		&vaultToken.ID,
		&vaultToken.PublicKey,
		&vaultToken.TokenID,
		&vaultToken.IssuedAt,
		&vaultToken.ExpiresAt,
		&vaultToken.IsRevoked,
		&vaultToken.LastUsedAt,
		&vaultToken.CreatedAt,
		&vaultToken.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &vaultToken, nil
}

func (p *PostgresBackend) RevokeVaultToken(ctx context.Context, tokenID string) error {
	query := `
		UPDATE vault_tokens
		SET is_revoked = true, updated_at = $1
		WHERE token_id = $2`

	_, err := p.pool.Exec(ctx, query, time.Now(), tokenID)
	return err
}

func (p *PostgresBackend) RevokeAllVaultTokens(ctx context.Context, publicKey string) error {
	query := `
		UPDATE vault_tokens
		SET is_revoked = true, updated_at = $1
		WHERE public_key = $2`

	_, err := p.pool.Exec(ctx, query, time.Now(), publicKey)
	return err
}

func (p *PostgresBackend) UpdateVaultTokenLastUsed(ctx context.Context, tokenID string) error {
	query := `
		UPDATE vault_tokens
		SET last_used_at = $1, updated_at = $2
		WHERE token_id = $3`

	now := time.Now()
	_, err := p.pool.Exec(ctx, query, now, now, tokenID)
	return err
}

func (p *PostgresBackend) GetActiveVaultTokens(ctx context.Context, publicKey string) ([]types.VaultToken, error) {
	query := `
		SELECT id, public_key, token_id, issued_at, expires_at, is_revoked, last_used_at, created_at, updated_at
		FROM vault_tokens
		WHERE public_key = $1
		AND is_revoked = false
		AND expires_at > $2
		ORDER BY issued_at DESC`

	rows, err := p.pool.Query(ctx, query, publicKey, time.Now())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tokens []types.VaultToken
	for rows.Next() {
		var token types.VaultToken
		err := rows.Scan(
			&token.ID,
			&token.PublicKey,
			&token.TokenID,
			&token.IssuedAt,
			&token.ExpiresAt,
			&token.IsRevoked,
			&token.LastUsedAt,
			&token.CreatedAt,
			&token.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, token)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tokens, nil
}
