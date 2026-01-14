package safety_pg

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Repo struct {
	pool *pgxpool.Pool
}

func NewRepo(pool *pgxpool.Pool) *Repo {
	return &Repo{pool: pool}
}

func (r *Repo) GetControlFlags(ctx context.Context, k1, k2 string) (map[string]bool, error) {
	result := make(map[string]bool, 2)

	const q = `
		SELECT key, enabled
		FROM control_flags
		WHERE key IN ($1, $2)
	`
	rows, err := r.pool.Query(ctx, q, k1, k2)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var k string
		var enabled bool
		scanErr := rows.Scan(&k, &enabled)
		if scanErr != nil {
			return nil, scanErr
		}
		result[k] = enabled
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}

	return result, nil
}
