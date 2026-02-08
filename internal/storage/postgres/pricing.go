package postgres

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/vultisig/verifier/types"
)

func (p *PostgresBackend) GetPricingByPluginId(ctx context.Context, pluginId string) ([]types.Pricing, error) {
	query := `SELECT pricings.* FROM pricings WHERE pricings.plugin_id = $1`

	rows, err := p.pool.Query(ctx, query, pluginId)
	if err != nil {
		return nil, err
	}

	pricing, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.Pricing])
	if err != nil {
		return nil, err
	}

	return pricing, nil
}

func (p *PostgresBackend) FindPricingById(ctx context.Context, id uuid.UUID) (*types.Pricing, error) {
	query := `SELECT * FROM pricings WHERE id = $1 LIMIT 1`

	rows, err := p.pool.Query(ctx, query, id)
	if err != nil {
		return nil, err
	}

	pricing, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[types.Pricing])
	if err != nil {
		return nil, err
	}

	return &pricing, nil
}

func (p *PostgresBackend) CreatePricing(ctx context.Context, pricingDto types.PricingCreateDto) (*types.Pricing, error) {
	query := `INSERT INTO pricings (plugin_id, type, frequency, amount, metric) 
	VALUES ($1, $2, $3, $4, $5) 
	RETURNING *`

	rows, err := p.pool.Query(ctx, query, pricingDto.PluginID, pricingDto.Type, pricingDto.Frequency, pricingDto.Amount, pricingDto.Metric)
	if err != nil {
		return nil, err
	}

	pricing, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[types.Pricing])
	if err != nil {
		return nil, err
	}

	return &pricing, nil
}

func (p *PostgresBackend) DeletePricingById(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM pricings WHERE id = $1`

	_, err := p.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}

	return nil
}
