package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/vultisig/verifier/internal/types"
)

const PLUGIN_RATING_TABLE = "plugin_ratings"

func (p *PostgresBackend) FindRatingByPluginId(ctx context.Context, dbTx pgx.Tx, pluginId string) ([]types.PluginRatingDto, error) {
	query := fmt.Sprintf(`
	SELECT rating_1_count, rating_2_count, rating_3_count, rating_4_count, rating_5_count
    FROM %s
    WHERE plugin_id = $1`, PLUGIN_RATING_TABLE)

	var rating1, rating2, rating3, rating4, rating5 int
	err := dbTx.QueryRow(ctx, query, pluginId).Scan(&rating1, &rating2, &rating3, &rating4, &rating5)
	if err != nil {
		if err == pgx.ErrNoRows {
			// No ratings yet, return empty array
			return []types.PluginRatingDto{}, nil
		}
		return nil, err
	}

	ratings := []types.PluginRatingDto{
		{Rating: 1, Count: rating1},
		{Rating: 2, Count: rating2},
		{Rating: 3, Count: rating3},
		{Rating: 4, Count: rating4},
		{Rating: 5, Count: rating5},
	}

	return ratings, nil
}

func (p *PostgresBackend) FindAvgRatingByPluginID(ctx context.Context, pluginID string) (types.PluginAvgRatingDto, error) {
	query := fmt.Sprintf(`
	SELECT avg_rating
    FROM %s
    WHERE plugin_id = $1`, PLUGIN_RATING_TABLE)

	var avgRating float64
	err := p.pool.QueryRow(ctx, query, pluginID).Scan(&avgRating)
	if err != nil {
		if err == pgx.ErrNoRows {
			// No ratings yet, return empty type.
			return types.PluginAvgRatingDto{}, nil
		}
		return types.PluginAvgRatingDto{}, err
	}

	resp := types.PluginAvgRatingDto{
		PluginID:  pluginID,
		RatingAvg: avgRating,
	}
	return resp, nil
}

func (p *PostgresBackend) CreateRatingForPlugin(ctx context.Context, dbTx pgx.Tx, pluginId string) error {
	ratingQuery := fmt.Sprintf(`INSERT INTO %s (plugin_id, avg_rating, total_ratings, rating_1_count, rating_2_count, rating_3_count, rating_4_count, rating_5_count)
	      VALUES ($1, 0, 0, 0, 0, 0, 0, 0)`, PLUGIN_RATING_TABLE)

	_, err := dbTx.Exec(ctx, ratingQuery, pluginId)
	if err != nil {
		return err
	}

	return nil
}

func (p *PostgresBackend) UpdateRatingForPlugin(ctx context.Context, dbTx pgx.Tx, pluginId string, reviewRating int) error {
	// Use UPSERT to atomically create the rating row if it doesn't exist
	upsertQuery := fmt.Sprintf(`
		INSERT INTO %s (plugin_id, avg_rating, total_ratings, rating_1_count, rating_2_count, rating_3_count, rating_4_count, rating_5_count, updated_at)
		VALUES ($1, 0, 0, 0, 0, 0, 0, 0, NOW())
		ON CONFLICT (plugin_id) DO NOTHING`, PLUGIN_RATING_TABLE)

	_, err := dbTx.Exec(ctx, upsertQuery, pluginId)
	if err != nil {
		return fmt.Errorf("failed to ensure rating row exists: %w", err)
	}

	// Now update the specific rating count and recalculate totals
	ratingQuery := fmt.Sprintf(`
	UPDATE %s
	SET rating_%d_count = rating_%d_count + 1,
	    total_ratings = total_ratings + 1,
	    avg_rating = (
	        (rating_1_count * 1 + rating_2_count * 2 + rating_3_count * 3 + rating_4_count * 4 + (rating_%d_count + 1) * %d)::DECIMAL 
	        / (total_ratings + 1)
	    ),
	    updated_at = NOW()
	WHERE plugin_id = $1`, PLUGIN_RATING_TABLE, reviewRating, reviewRating, reviewRating, reviewRating)

	ct, err := dbTx.Exec(ctx, ratingQuery, pluginId)
	if err != nil {
		return fmt.Errorf("failed to update rating: %w", err)
	}

	if ct.RowsAffected() == 0 {
		return fmt.Errorf("rating update affected 0 rows for plugin_id=%s rating=%d", pluginId, reviewRating)
	}

	return nil
}

func (p *PostgresBackend) ChangeRatingForPlugin(ctx context.Context, dbTx pgx.Tx, pluginId string, oldRating int, newRating int) error {
	// Use UPSERT to atomically create the rating row if it doesn't exist
	upsertQuery := fmt.Sprintf(`
		INSERT INTO %s (plugin_id, avg_rating, total_ratings, rating_1_count, rating_2_count, rating_3_count, rating_4_count, rating_5_count, updated_at)
		VALUES ($1, 0, 0, 0, 0, 0, 0, 0, NOW())
		ON CONFLICT (plugin_id) DO NOTHING`, PLUGIN_RATING_TABLE)

	_, err := dbTx.Exec(ctx, upsertQuery, pluginId)
	if err != nil {
		return fmt.Errorf("failed to ensure rating row exists: %w", err)
	}

	// Update both old and new rating counts without changing total count
	ratingQuery := fmt.Sprintf(`
	UPDATE %s
	SET rating_%d_count = GREATEST(rating_%d_count - 1, 0),
	    rating_%d_count = rating_%d_count + 1,
	    avg_rating = (
	        (GREATEST(rating_1_count - CASE WHEN %d = 1 THEN 1 ELSE 0 END, 0) * 1 + 
	         GREATEST(rating_2_count - CASE WHEN %d = 2 THEN 1 ELSE 0 END, 0) * 2 + 
	         GREATEST(rating_3_count - CASE WHEN %d = 3 THEN 1 ELSE 0 END, 0) * 3 + 
	         GREATEST(rating_4_count - CASE WHEN %d = 4 THEN 1 ELSE 0 END, 0) * 4 + 
	         GREATEST(rating_5_count - CASE WHEN %d = 5 THEN 1 ELSE 0 END, 0) * 5 +
	         (CASE WHEN %d = 1 THEN 1 ELSE 0 END) * 1 +
	         (CASE WHEN %d = 2 THEN 1 ELSE 0 END) * 2 +
	         (CASE WHEN %d = 3 THEN 1 ELSE 0 END) * 3 +
	         (CASE WHEN %d = 4 THEN 1 ELSE 0 END) * 4 +
	         (CASE WHEN %d = 5 THEN 1 ELSE 0 END) * 5)::DECIMAL 
	        / GREATEST(total_ratings, 1)
	    ),
	    updated_at = NOW()
	WHERE plugin_id = $1`, PLUGIN_RATING_TABLE, oldRating, oldRating, newRating, newRating,
		oldRating, oldRating, oldRating, oldRating, oldRating,
		newRating, newRating, newRating, newRating, newRating)

	ct, err := dbTx.Exec(ctx, ratingQuery, pluginId)
	if err != nil {
		return fmt.Errorf("failed to update rating: %w", err)
	}

	if ct.RowsAffected() == 0 {
		return fmt.Errorf("rating update affected 0 rows for plugin_id=%s", pluginId)
	}

	return nil
}
