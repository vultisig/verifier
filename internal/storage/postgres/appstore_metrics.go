package postgres

import (
	"context"
	"fmt"
)

func (p *PostgresBackend) GetInstallationsByPlugin(ctx context.Context) (map[string]int64, error) {
	query := `
		SELECT plugin_id, COUNT(public_key) as count
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

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("error iterating installation rows: %w", err)
	}

	return result, nil
}

func (p *PostgresBackend) GetPoliciesByPlugin(ctx context.Context) (map[string]int64, error) {
	query := `
		SELECT plugin_id, COUNT(*) as count
		FROM plugin_policies
		WHERE active = true AND deleted = false
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

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("error iterating policy rows: %w", err)
	}

	return result, nil
}

func (p *PostgresBackend) GetFeesByPlugin(ctx context.Context) (map[string]int64, error) {
	query := `
		SELECT plugin_id, SUM(amount) as total
		FROM fees
		WHERE transaction_type = 'debit' AND plugin_id IS NOT NULL
		GROUP BY plugin_id
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

	err = rows.Err()
	if err != nil {
		return nil, fmt.Errorf("error iterating fee rows: %w", err)
	}

	return result, nil
}
