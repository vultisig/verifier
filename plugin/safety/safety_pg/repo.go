package safety_pg

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vultisig/verifier/plugin/safety"
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

func (r *Repo) UpsertControlFlags(ctx context.Context, flags []safety.ControlFlag) error {
	for _, flag := range flags {
		_, err := r.pool.Exec(ctx, `
			INSERT INTO control_flags (key, enabled, updated_at)
			VALUES ($1, $2, NOW())
			ON CONFLICT (key) DO UPDATE
			SET enabled = EXCLUDED.enabled, updated_at = NOW()
		`, flag.Key, flag.Enabled)
		if err != nil {
			return err
		}
	}
	return nil
}
