package postgres

import (
	"context"
	"fmt"

	"github.com/vultisig/verifier/types"
)

func (p *PostgresBackend) IsOwner(ctx context.Context, pluginID types.PluginID, publicKey string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM plugin_owners WHERE plugin_id = $1 AND public_key = $2 AND active = TRUE)`
	var exists bool
	err := p.pool.QueryRow(ctx, query, pluginID, publicKey).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check ownership: %w", err)
	}
	return exists, nil
}

func (p *PostgresBackend) GetPluginsByOwner(ctx context.Context, publicKey string) ([]types.PluginID, error) {
	query := `SELECT plugin_id FROM plugin_owners WHERE public_key = $1 AND active = TRUE`
	rows, err := p.pool.Query(ctx, query, publicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get plugins by owner: %w", err)
	}
	defer rows.Close()

	var pluginIDs []types.PluginID
	for rows.Next() {
		var id types.PluginID
		err := rows.Scan(&id)
		if err != nil {
			return nil, fmt.Errorf("failed to scan plugin id: %w", err)
		}
		pluginIDs = append(pluginIDs, id)
	}

	return pluginIDs, nil
}

func (p *PostgresBackend) AddOwner(ctx context.Context, pluginID types.PluginID, publicKey, addedVia, addedBy string) error {
	query := `
		INSERT INTO plugin_owners (plugin_id, public_key, active, added_via, added_by_public_key)
		VALUES ($1, $2, TRUE, $3, $4)
		ON CONFLICT (plugin_id, public_key)
		DO UPDATE SET active = TRUE, updated_at = NOW(), added_via = EXCLUDED.added_via, added_by_public_key = EXCLUDED.added_by_public_key
	`
	_, err := p.pool.Exec(ctx, query, pluginID, publicKey, addedVia, addedBy)
	if err != nil {
		return fmt.Errorf("failed to add owner: %w", err)
	}
	return nil
}

func (p *PostgresBackend) DeactivateOwner(ctx context.Context, pluginID types.PluginID, publicKey string) error {
	query := `UPDATE plugin_owners SET active = FALSE, updated_at = NOW() WHERE plugin_id = $1 AND public_key = $2`
	_, err := p.pool.Exec(ctx, query, pluginID, publicKey)
	if err != nil {
		return fmt.Errorf("failed to deactivate owner: %w", err)
	}
	return nil
}
