package postgres

import (
	"context"
	"fmt"
)

func (p *PostgresBackend) GetInstallationsByPlugin(ctx context.Context) (map[string]int64, error) {
	query := `
		SELECT plugin_id, COUNT(*) as count
		FROM plugin_installations
		GROUP BY plugin_id
	`

	rows, err := p.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query installations: %w", err)
	}
	defer rows.Close()

	result := make(map[string]int64)
	for rows.Next() {
		var pluginID string
		var count int64
		err = rows.Scan(&pluginID, &count)
		if err != nil {
			return nil, fmt.Errorf("failed to scan installation row: %w", err)
		}
		result[pluginID] = count
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating installations: %w", err)
	}

	return result, nil
}

func (p *PostgresBackend) GetPoliciesByPlugin(ctx context.Context) (map[string]int64, error) {
	query := `
		SELECT plugin_id, COUNT(*) as count
		FROM plugin_policies
		WHERE is_active = true
		GROUP BY plugin_id
	`

	rows, err := p.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query policies: %w", err)
	}
	defer rows.Close()

	result := make(map[string]int64)
	for rows.Next() {
		var pluginID string
		var count int64
		err = rows.Scan(&pluginID, &count)
		if err != nil {
			return nil, fmt.Errorf("failed to scan policy row: %w", err)
		}
		result[pluginID] = count
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating policies: %w", err)
	}

	return result, nil
}

func (p *PostgresBackend) GetFeesByPlugin(ctx context.Context) (map[string]int64, error) {
	query := `
		SELECT pp.plugin_id, COALESCE(SUM(f.value), 0) as total
		FROM fees f
		JOIN plugin_policies pp ON f.policy_id = pp.id
		WHERE f.is_collected = true
		GROUP BY pp.plugin_id
	`

	rows, err := p.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query fees: %w", err)
	}
	defer rows.Close()

	result := make(map[string]int64)
	for rows.Next() {
		var pluginID string
		var total int64
		err = rows.Scan(&pluginID, &total)
		if err != nil {
			return nil, fmt.Errorf("failed to scan fee row: %w", err)
		}
		result[pluginID] = total
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating fees: %w", err)
	}

	return result, nil
}
