package postgres

import (
	"context"
	"fmt"
)

func (p *PostgresBackend) IsProposedPluginApproved(ctx context.Context, pluginID string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM proposed_plugins WHERE plugin_id = $1 AND status = 'approved')`
	var exists bool
	err := p.pool.QueryRow(ctx, query, pluginID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check proposed plugin: %w", err)
	}
	return exists, nil
}
