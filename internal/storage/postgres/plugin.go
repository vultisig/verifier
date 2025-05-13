package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/vultisig/verifier/common"
	"github.com/vultisig/verifier/internal/types"
)

const PLUGINS_TABLE = "plugins"

func (p *PostgresBackend) FindPluginById(ctx context.Context, id uuid.UUID) (*types.Plugin, error) {
	query := fmt.Sprintf(`SELECT * FROM %s WHERE id = $1 LIMIT 1;`, PLUGINS_TABLE)

	rows, err := p.pool.Query(ctx, query, id)
	if err != nil {
		return nil, err
	}

	plugin, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[types.Plugin])
	if err != nil {
		return nil, err
	}

	return &plugin, nil
}

func (p *PostgresBackend) FindPlugins(ctx context.Context, take int, skip int, sort string) (types.PluginsDto, error) {
	if p.pool == nil {
		return types.PluginsDto{}, fmt.Errorf("database pool is nil")
	}

	orderBy, orderDirection := common.GetSortingCondition(sort)

	query := fmt.Sprintf(`
		SELECT *, COUNT(*) OVER() AS total_count
		FROM %s 
		ORDER BY %s %s
		LIMIT $1 OFFSET $2`, PLUGINS_TABLE, orderBy, orderDirection)

	rows, err := p.pool.Query(ctx, query, take, skip)
	if err != nil {
		return types.PluginsDto{}, err
	}

	defer rows.Close()

	var plugins []types.Plugin
	var totalCount int

	for rows.Next() {
		var plugin types.Plugin

		err := rows.Scan(
			&plugin.ID,
			&plugin.CreatedAt,
			&plugin.UpdatedAt,
			&plugin.Type,
			&plugin.Title,
			&plugin.Description,
			&plugin.Metadata,
			&plugin.ServerEndpoint,
			&plugin.PricingID,
			&totalCount,
		)
		if err != nil {
			return types.PluginsDto{}, err
		}

		plugins = append(plugins, plugin)
	}

	pluginsDto := types.PluginsDto{
		Plugins:    plugins,
		TotalCount: totalCount,
	}

	return pluginsDto, nil
}

func (p *PostgresBackend) CreatePlugin(ctx context.Context, pluginDto types.PluginCreateDto) (*types.Plugin, error) {
	query := fmt.Sprintf(`INSERT INTO %s (
		type,
		title,
		description,
		metadata,
		server_endpoint,
		pricing_id
	) VALUES (
		@Type,
		@Title,
		@Description,
		@Metadata,
		@ServerEndpoint,
		@PricingID
	) RETURNING id;`, PLUGINS_TABLE)
	args := pgx.NamedArgs{
		"Type":           pluginDto.Type,
		"Title":          pluginDto.Title,
		"Description":    pluginDto.Description,
		"Metadata":       pluginDto.Metadata,
		"ServerEndpoint": pluginDto.ServerEndpoint,
		"PricingID":      pluginDto.PricingID,
	}

	var createdId uuid.UUID
	err := p.pool.QueryRow(ctx, query, args).Scan(&createdId)
	if err != nil {
		return nil, err
	}

	return p.FindPluginById(ctx, createdId)
}

func (p *PostgresBackend) UpdatePlugin(ctx context.Context, id uuid.UUID, updates types.PluginUpdateDto) (*types.Plugin, error) {
	query := fmt.Sprintf(`UPDATE plugins SET title = @Title, description = @Description, metadata = @Metadata, server_endpoint = @ServerEndpoint, pricing_id = @PricingID WHERE id = @id;`)
	args := pgx.NamedArgs{
		"Title":          updates.Title,
		"Description":    updates.Description,
		"Metadata":       updates.Metadata,
		"ServerEndpoint": updates.ServerEndpoint,
		"PricingID":      updates.PricingID,
		"id":             id,
	}
	_, err := p.pool.Exec(ctx, query, args)
	if err != nil {
		return nil, err
	}

	return p.FindPluginById(ctx, id)
}

func (p *PostgresBackend) DeletePluginById(ctx context.Context, id uuid.UUID) error {
	query := fmt.Sprintf(`DELETE FROM %s WHERE id = $1;`, PLUGINS_TABLE)

	_, err := p.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}

	return nil
}
