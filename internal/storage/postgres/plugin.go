package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	itypes "github.com/vultisig/verifier/internal/types"
	"github.com/vultisig/verifier/types"
	"github.com/vultisig/vultisig-go/common"
)

const PLUGINS_TABLE = "plugins"
const PLUGIN_TAGS_TABLE = "plugin_tags"
const REVIEWS_TABLE = "reviews"

// This is needed as the plugins table is left joined with the pricings table, and when a plugin is free (i.e zero related pricing records) it tries to scan null into a non nullable struct
type nullablePricing struct {
	ID        *uuid.UUID
	Type      *string
	Frequency *string
	Amount    *uint64
	Asset     *string
	Metric    *string
	PluginID  *string
	CreatedAt *time.Time
	UpdatedAt *time.Time
}

func convertNullablePricing(np *nullablePricing) *types.Pricing {
	if np.ID == nil || np.Type == nil || np.Amount == nil ||
		np.Asset == nil || np.Metric == nil || np.PluginID == nil ||
		np.CreatedAt == nil || np.UpdatedAt == nil {
		return nil
	}

	var frequency *types.PricingFrequency
	if np.Frequency != nil {
		freq := types.PricingFrequency(*np.Frequency)
		frequency = &freq
	}

	return &types.Pricing{
		ID:        *np.ID,
		Type:      types.PricingType(*np.Type),
		Frequency: frequency,
		Amount:    *np.Amount,
		Asset:     types.PricingAsset(*np.Asset),
		Metric:    types.PricingMetric(*np.Metric),
		PluginID:  types.PluginID(*np.PluginID),
		CreatedAt: *np.CreatedAt,
		UpdatedAt: *np.UpdatedAt,
	}
}

func (p *PostgresBackend) collectPlugins(rows pgx.Rows) ([]itypes.Plugin, error) {
	defer rows.Close()

	// Use a map to group plugins by ID and collect their pricing records
	pluginMap := make(map[types.PluginID]*itypes.Plugin)
	pluginIds := []types.PluginID{}

	for rows.Next() {
		var plugin itypes.Plugin
		var tagId *string
		var tagName *string
		var tagCreatedAt *time.Time

		nullablePricing := &nullablePricing{}

		err := rows.Scan(
			&plugin.ID,
			&plugin.Title,
			&plugin.Description,
			&plugin.ServerEndpoint,
			&plugin.Category,
			&plugin.CreatedAt,
			&plugin.UpdatedAt,
			&tagId,
			&tagName,
			&tagCreatedAt,
			&nullablePricing.ID,
			&nullablePricing.Type,
			&nullablePricing.Frequency,
			&nullablePricing.Amount,
			&nullablePricing.Asset,
			&nullablePricing.Metric,
			&nullablePricing.PluginID,
			&nullablePricing.CreatedAt,
			&nullablePricing.UpdatedAt,
		)

		if err != nil {
			return nil, err
		}

		// Check if we've seen this plugin before
		if existingPlugin, exists := pluginMap[plugin.ID]; exists {
			// Plugin already exists, just add the pricing record if it's valid
			if nullablePricing.ID != nil {
				if pricing := convertNullablePricing(nullablePricing); pricing != nil {
					existingPlugin.Pricing = append(existingPlugin.Pricing, *pricing)
				}
			}
		} else {
			// New plugin, initialize the pricing slice
			plugin.Pricing = make([]types.Pricing, 0)
			if nullablePricing.ID != nil {
				if pricing := convertNullablePricing(nullablePricing); pricing != nil {
					plugin.Pricing = append(plugin.Pricing, *pricing)
				}
			}
			pluginMap[plugin.ID] = &plugin
			pluginIds = append(pluginIds, plugin.ID)
		}
	}

	// Convert map back to slice
	plugins := make([]itypes.Plugin, 0, len(pluginMap))
	for _, pluginId := range pluginIds {
		plugins = append(plugins, *pluginMap[pluginId])
	}

	return plugins, nil
}

func (p *PostgresBackend) FindPluginById(ctx context.Context, dbTx pgx.Tx, id types.PluginID) (*itypes.Plugin, error) {
	query := fmt.Sprintf(
		`SELECT p.*, t.*, pr.*
		FROM %s p
		LEFT JOIN plugin_tags pt ON p.id = pt.plugin_id
		LEFT JOIN tags t ON pt.tag_id = t.id
		LEFT JOIN pricings pr ON p.id = pr.plugin_id
		WHERE p.id = $1;`,
		PLUGINS_TABLE,
	)

	var rows pgx.Rows
	var err error
	if dbTx != nil {
		rows, err = dbTx.Query(ctx, query, id)
	} else {
		rows, err = p.pool.Query(ctx, query, id)
	}
	if err != nil {
		return nil, err
	}

	plugins, err := p.collectPlugins(rows)
	if err != nil {
		return nil, err
	}

	if len(plugins) == 0 {
		return nil, fmt.Errorf("plugin not found")
	}

	return &plugins[0], nil
}

func (p *PostgresBackend) FindPlugins(
	ctx context.Context,
	filters itypes.PluginFilters,
	take int,
	skip int,
	sort string,
) (*itypes.PluginsPaginatedList, error) {
	if p.pool == nil {
		return nil, fmt.Errorf("database pool is nil")
	}

	joinQuery := fmt.Sprintf(`
		FROM %s p
		LEFT JOIN plugin_tags pt ON p.id = pt.plugin_id
		LEFT JOIN tags t ON pt.tag_id = t.id
		LEFT JOIN pricings pr ON p.id = pr.plugin_id`,
		PLUGINS_TABLE,
	)

	query := `SELECT p.*, t.*, pr.*` + joinQuery
	queryTotal := `SELECT COUNT(DISTINCT p.id) as total_count` + joinQuery

	var args []any
	var argsTotal []any
	currentArgNumber := 1

	// filters
	filterClause := "WHERE"
	if filters.Term != nil {
		queryFilter := fmt.Sprintf(
			` %s (p.title ILIKE $%d OR p.description ILIKE $%d)`,
			filterClause,
			currentArgNumber,
			currentArgNumber+1,
		)
		filterClause = "AND"
		currentArgNumber += 2

		term := "%" + *filters.Term + "%"
		args = append(args, term, term)
		argsTotal = append(argsTotal, term, term)

		query += queryFilter
		queryTotal += queryFilter
	}

	if filters.TagID != nil {
		queryFilter := fmt.Sprintf(
			` %s p.id IN (
				SELECT pti.plugin_id
    		FROM plugin_tags pti
    		JOIN tags ti ON pti.tag_id = ti.id
    		WHERE ti.id = $%d
			)`,
			filterClause,
			currentArgNumber,
		)

		queryFilterTotal := fmt.Sprintf(
			` %s t.id = $%d`,
			filterClause,
			currentArgNumber,
		)
		filterClause = "AND"
		currentArgNumber += 1

		args = append(args, *filters.TagID)
		argsTotal = append(argsTotal, *filters.TagID)

		query += queryFilter
		queryTotal += queryFilterTotal
	}

	if filters.CategoryID != nil {
		queryFilter := fmt.Sprintf(
			` %s p.category = $%d`,
			filterClause,
			currentArgNumber,
		)
		filterClause = "AND"
		currentArgNumber += 1

		args = append(args, filters.CategoryID)
		argsTotal = append(argsTotal, filters.CategoryID)

		query += queryFilter
		queryTotal += queryFilter
	}

	// pagination
	queryOrderPaginate := fmt.Sprintf(
		` ORDER BY p.id LIMIT $%d OFFSET $%d;`,
		currentArgNumber,
		currentArgNumber+1,
	)

	args = append(args, take, skip)
	query += queryOrderPaginate

	queryTotal += ";"

	// execute
	rows, err := p.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}

	plugins, err := p.collectPlugins(rows)
	if err != nil {
		return nil, err
	}

	// execute total results count
	var totalCount int
	err = p.pool.QueryRow(ctx, queryTotal, argsTotal...).Scan(&totalCount)
	if err != nil {
		// exactly 1 row expected, if no results return empty list
		if errors.Is(err, pgx.ErrNoRows) {
			return &itypes.PluginsPaginatedList{
				Plugins:    plugins,
				TotalCount: 0,
			}, nil
		}
		return nil, err
	}

	pluginsList := itypes.PluginsPaginatedList{
		Plugins:    plugins,
		TotalCount: totalCount,
	}

	return &pluginsList, nil
}

func (p *PostgresBackend) FindReviewById(ctx context.Context, db pgx.Tx, id string) (*itypes.ReviewDto, error) {
	query := fmt.Sprintf(`SELECT id, plugin_id, public_key, rating, comment, created_at FROM %s WHERE id = $1 LIMIT 1;`, REVIEWS_TABLE)

	var reviewDto itypes.ReviewDto
	var err error

	if db != nil {
		err = db.QueryRow(ctx, query, id).Scan(
			&reviewDto.ID,
			&reviewDto.PluginId,
			&reviewDto.Address,
			&reviewDto.Rating,
			&reviewDto.Comment,
			&reviewDto.CreatedAt,
		)
	} else {
		err = p.pool.QueryRow(ctx, query, id).Scan(
			&reviewDto.ID,
			&reviewDto.PluginId,
			&reviewDto.Address,
			&reviewDto.Rating,
			&reviewDto.Comment,
			&reviewDto.CreatedAt,
		)
	}

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("review not found")
		}
		return nil, err
	}

	return &reviewDto, nil
}

func (p *PostgresBackend) FindReviews(ctx context.Context, pluginId string, skip int, take int, sort string) (itypes.ReviewsDto, error) {
	if p.pool == nil {
		return itypes.ReviewsDto{}, fmt.Errorf("database pool is nil")
	}

	allowedSortingColumns := map[string]bool{"created_at": true}
	orderBy, orderDirection := common.GetSortingCondition(sort, allowedSortingColumns)

	query := fmt.Sprintf(`
		SELECT id, plugin_id, public_key, rating, comment, created_at, COUNT(*) OVER() AS total_count
		FROM %s
		WHERE plugin_id = $1
		ORDER BY %s %s
		LIMIT $2 OFFSET $3`, REVIEWS_TABLE, orderBy, orderDirection)

	rows, err := p.pool.Query(ctx, query, pluginId, take, skip)
	if err != nil {
		return itypes.ReviewsDto{}, err
	}

	defer rows.Close()

	var reviews []itypes.Review
	var totalCount int

	for rows.Next() {
		var review itypes.Review

		err := rows.Scan(
			&review.ID,
			&review.PluginId,
			&review.Address,
			&review.Rating,
			&review.Comment,
			&review.CreatedAt,
			&totalCount,
		)
		if err != nil {
			return itypes.ReviewsDto{}, err
		}

		reviews = append(reviews, review)
	}

	pluginsDto := itypes.ReviewsDto{
		Reviews:    reviews,
		TotalCount: totalCount,
	}

	return pluginsDto, nil
}

func (p *PostgresBackend) FindReviewByUserAndPlugin(ctx context.Context, dbTx pgx.Tx, pluginId string, userAddress string) (*itypes.ReviewDto, error) {
	query := fmt.Sprintf(`SELECT id, plugin_id, public_key, rating, comment, created_at FROM %s WHERE plugin_id = $1 AND LOWER(public_key) = LOWER($2) LIMIT 1;`, REVIEWS_TABLE)

	var reviewDto itypes.ReviewDto
	var err error

	if dbTx != nil {
		err = dbTx.QueryRow(ctx, query, pluginId, userAddress).Scan(
			&reviewDto.ID,
			&reviewDto.PluginId,
			&reviewDto.Address,
			&reviewDto.Rating,
			&reviewDto.Comment,
			&reviewDto.CreatedAt,
		)
	} else {
		err = p.pool.QueryRow(ctx, query, pluginId, userAddress).Scan(
			&reviewDto.ID,
			&reviewDto.PluginId,
			&reviewDto.Address,
			&reviewDto.Rating,
			&reviewDto.Comment,
			&reviewDto.CreatedAt,
		)
	}

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // No existing review found
		}
		return nil, err
	}

	return &reviewDto, nil
}

func (p *PostgresBackend) UpdateReview(ctx context.Context, dbTx pgx.Tx, reviewId string, reviewDto itypes.ReviewCreateDto) error {
	query := fmt.Sprintf(`UPDATE %s SET rating = $1, comment = $2, updated_at = NOW() WHERE id = $3`, REVIEWS_TABLE)

	ct, err := dbTx.Exec(ctx, query, reviewDto.Rating, reviewDto.Comment, reviewId)
	if err != nil {
		return fmt.Errorf("failed to update review: %w", err)
	}

	if ct.RowsAffected() == 0 {
		return fmt.Errorf("review not found with id: %s", reviewId)
	}

	return nil
}

func (p *PostgresBackend) CreateReview(ctx context.Context, dbTx pgx.Tx, reviewDto itypes.ReviewCreateDto, pluginId string) (string, error) {
	// Fix: Use public_key instead of address to match the database schema
	columns := []string{"public_key", "rating", "comment", "plugin_id", "created_at"}
	argNames := []string{"@PublicKey", "@Rating", "@Comment", "@PluginId", "@CreatedAt"}
	args := pgx.NamedArgs{
		"PublicKey": reviewDto.Address, // Map Address field to public_key column
		"Rating":    reviewDto.Rating,
		"Comment":   reviewDto.Comment,
		"PluginId":  pluginId,
		"CreatedAt": time.Now(),
	}

	query := fmt.Sprintf(
		`INSERT INTO reviews (%s) VALUES (%s) RETURNING id;`,
		strings.Join(columns, ", "),
		strings.Join(argNames, ", "),
	)

	var createdId string
	err := dbTx.QueryRow(ctx, query, args).Scan(&createdId)
	if err != nil {
		return "", fmt.Errorf("failed to create review: %w", err)
	}

	return createdId, nil
}
