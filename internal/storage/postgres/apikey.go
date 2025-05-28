package postgres

import (
	"context"
	"fmt"

	"github.com/vultisig/verifier/internal/types"
)

func (p *PostgresBackend) GetAPIKey(ctx context.Context, apiKey string) (*types.APIKey, error) {
	query := `
		SELECT 
			id, 
			plugin_id, 
			apikey, 
			status, 
			expires_at,
		FROM plugin_apikeys
		WHERE apikey = $1
	`
	var key types.APIKey
	if err := p.pool.QueryRow(ctx, query, apiKey).Scan(
		&key.ID,
		&key.PluginID,
		&key.ApiKey,
		&key.Status,
		&key.ExpiresAt); err != nil {
		return nil, fmt.Errorf("fail to get API key: %w", err)
	}
	return &key, nil
}
