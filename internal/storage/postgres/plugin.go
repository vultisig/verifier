package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/jackc/pgx/v5"

	"github.com/vultisig/verifier/common"
	"github.com/vultisig/verifier/internal/types"
	ptypes "github.com/vultisig/verifier/types"
)

const PLUGINS_TABLE = "plugins"

func (p *PostgresBackend) FindPluginById(ctx context.Context, id ptypes.PluginID) (*types.Plugin, error) {
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

	var createdId ptypes.PluginID
	err := p.pool.QueryRow(ctx, query, args).Scan(&createdId)
	if err != nil {
		return nil, err
	}

	return p.FindPluginById(ctx, createdId)
}

func (p *PostgresBackend) UpdatePlugin(ctx context.Context, id ptypes.PluginID, updates types.PluginUpdateDto) (*types.Plugin, error) {
	t := reflect.TypeOf(updates)
	v := reflect.ValueOf(updates)
	numFields := t.NumField()

	query := fmt.Sprintf(`UPDATE %s SET `, PLUGINS_TABLE)

	args := pgx.NamedArgs{
		"id": id,
	}

	// iterate over dto props and assign non-empty for update
	var updateStatements []string
	for i := 0; i < numFields; i++ {
		field := t.Field(i) // field metadata
		value := v.Field(i) // field value

		// filter out json undefined values
		if !value.IsNil() {
			// get db field name (same as it is defined in the json input)
			fieldName := field.Tag.Get("json")
			if fieldName == "" {
				// fallback to prop name
				fieldName = field.Name
			}

			// get value from dto reference
			var fieldValue interface{}
			if field.Type == reflect.TypeOf((*json.RawMessage)(nil)) {
				// keep as reference to []byte
				fieldValue = value.Interface().(*json.RawMessage)
			} else {
				// dereference
				fieldValue = value.Elem().Interface()
			}

			updateStatements = append(updateStatements, fmt.Sprintf("%s = @%s", fieldName, fieldName))
			args[fieldName] = fieldValue
		}
	}

	query += strings.Join(updateStatements, ", ") + ` WHERE id = @id`

	_, err := p.pool.Exec(ctx, query, args)
	if err != nil {
		return nil, err
	}

	return p.FindPluginById(ctx, id)
}

func (p *PostgresBackend) DeletePluginById(ctx context.Context, id ptypes.PluginID) error {
	query := fmt.Sprintf(`DELETE FROM %s WHERE id = $1;`, PLUGINS_TABLE)

	_, err := p.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}

	return nil
}
