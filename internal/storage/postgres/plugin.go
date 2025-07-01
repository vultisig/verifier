package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/vultisig/verifier/common"
	"github.com/vultisig/verifier/internal/types"
	ptypes "github.com/vultisig/verifier/types"
)

const PLUGINS_TABLE = "plugins"
const PLUGIN_TAGS_TABLE = "plugin_tags"
const REVIEWS_TABLE = "reviews"

func (p *PostgresBackend) collectPlugins(rows pgx.Rows) ([]types.Plugin, error) {
	defer rows.Close()

	var plugins []types.Plugin
	for rows.Next() {
		var plugin types.Plugin
		var tagId *string
		var tagName *string
		var tagCreatedAt *time.Time

		err := rows.Scan(
			&plugin.ID,
			&plugin.Title,
			&plugin.Description,
			&plugin.ServerEndpoint,
			&plugin.PricingID,
			&plugin.Category,
			&plugin.CreatedAt,
			&plugin.UpdatedAt,
			&tagId,
			&tagName,
			&tagCreatedAt,
		)

		if err != nil {
			return plugins, err
		}

		plugins = append(plugins, plugin)
	}

	return plugins, nil
}

func (p *PostgresBackend) FindPluginById(ctx context.Context, dbTx pgx.Tx, id ptypes.PluginID) (*types.Plugin, error) {
	query := fmt.Sprintf(
		`SELECT p.*, t.*
		FROM %s p
		LEFT JOIN plugin_tags pt ON p.id = pt.plugin_id
		LEFT JOIN tags t ON pt.tag_id = t.id
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
	filters types.PluginFilters,
	take int,
	skip int,
	sort string,
) (types.PluginsPaginatedList, error) {
	if p.pool == nil {
		return types.PluginsPaginatedList{}, fmt.Errorf("database pool is nil")
	}

	allowedSortingColumns := map[string]bool{"updated_at": true, "created_at": true, "title": true}
	orderBy, orderDirection := common.GetSortingCondition(sort, allowedSortingColumns)

	joinQuery := fmt.Sprintf(`
		FROM %s p
		LEFT JOIN plugin_tags pt ON p.id = pt.plugin_id
		LEFT JOIN tags t ON pt.tag_id = t.id`,
		PLUGINS_TABLE,
	)

	query := `SELECT p.*, t.*` + joinQuery
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
			` %s p.category_id = $%d`,
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
		` ORDER BY p.%s %s LIMIT $%d OFFSET $%d;`,
		pgx.Identifier{orderBy}.Sanitize(),
		orderDirection,
		currentArgNumber,
		currentArgNumber+1,
	)
	args = append(args, take, skip)
	query += queryOrderPaginate

	queryTotal += ";"

	fmt.Println(query)

	// execute
	rows, err := p.pool.Query(ctx, query, args...)
	if err != nil {
		return types.PluginsPaginatedList{}, err
	}

	plugins, err := p.collectPlugins(rows)
	if err != nil {
		return types.PluginsPaginatedList{}, err
	}

	// execute total results count
	var totalCount int
	err = p.pool.QueryRow(ctx, queryTotal, argsTotal...).Scan(&totalCount)
	if err != nil {
		// exactly 1 row expected, if no results return empty list
		if errors.Is(err, pgx.ErrNoRows) {
			return types.PluginsPaginatedList{
				Plugins:    plugins,
				TotalCount: 0,
			}, nil
		}
		return types.PluginsPaginatedList{}, err
	}

	pluginsList := types.PluginsPaginatedList{
		Plugins:    plugins,
		TotalCount: totalCount,
	}

	return pluginsList, nil
}

func (p *PostgresBackend) FindReviewById(ctx context.Context, db pgx.Tx, id string) (*types.ReviewDto, error) {
	query := fmt.Sprintf(`SELECT id, plugin_id, public_key, rating, comment, created_at FROM %s WHERE id = $1 LIMIT 1;`, REVIEWS_TABLE)

	var reviewDto types.ReviewDto
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

func (p *PostgresBackend) FindReviews(ctx context.Context, pluginId string, skip int, take int, sort string) (types.ReviewsDto, error) {
	if p.pool == nil {
		return types.ReviewsDto{}, fmt.Errorf("database pool is nil")
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
		return types.ReviewsDto{}, err
	}

	defer rows.Close()

	var reviews []types.Review
	var totalCount int

	for rows.Next() {
		var review types.Review

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
			return types.ReviewsDto{}, err
		}

		reviews = append(reviews, review)
	}

	pluginsDto := types.ReviewsDto{
		Reviews:    reviews,
		TotalCount: totalCount,
	}

	return pluginsDto, nil
}

func (p *PostgresBackend) FindReviewByUserAndPlugin(ctx context.Context, dbTx pgx.Tx, pluginId string, userAddress string) (*types.ReviewDto, error) {
	query := fmt.Sprintf(`SELECT id, plugin_id, public_key, rating, comment, created_at FROM %s WHERE plugin_id = $1 AND LOWER(public_key) = LOWER($2) LIMIT 1;`, REVIEWS_TABLE)

	var reviewDto types.ReviewDto
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

func (p *PostgresBackend) UpdateReview(ctx context.Context, dbTx pgx.Tx, reviewId string, reviewDto types.ReviewCreateDto) error {
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

func (p *PostgresBackend) CreateReview(ctx context.Context, dbTx pgx.Tx, reviewDto types.ReviewCreateDto, pluginId string) (string, error) {
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
