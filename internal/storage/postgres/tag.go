package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/vultisig/verifier/internal/types"
)

const TAGS_TABLE = "tags"

func (p *PostgresBackend) FindTags(ctx context.Context) ([]types.Tag, error) {
	query := fmt.Sprintf(`SELECT * FROM %s`, TAGS_TABLE)

	rows, err := p.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}

	tags, err := pgx.CollectRows(rows, pgx.RowToStructByName[types.Tag])
	if err != nil {
		return nil, err
	}

	return tags, nil
}

func (p *PostgresBackend) FindTagById(ctx context.Context, id string) (*types.Tag, error) {
	query := fmt.Sprintf(`SELECT * FROM %s WHERE id = $1 LIMIT 1;`, TAGS_TABLE)

	rows, err := p.pool.Query(ctx, query, id)
	if err != nil {
		return nil, err
	}

	tag, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[types.Tag])
	if err != nil {
		return nil, err
	}

	return &tag, nil
}

func (p *PostgresBackend) FindTagByName(ctx context.Context, name string) (*types.Tag, error) {
	query := fmt.Sprintf(`SELECT * FROM %s WHERE name = $1 LIMIT 1;`, TAGS_TABLE)

	rows, err := p.pool.Query(ctx, query, name)
	if err != nil {
		return nil, err
	}

	tag, err := pgx.CollectOneRow(rows, pgx.RowToStructByName[types.Tag])
	if err != nil {
		return nil, err
	}

	return &tag, nil
}
