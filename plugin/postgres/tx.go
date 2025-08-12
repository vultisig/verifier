package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Caller interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults
	CopyFrom(
		ctx context.Context,
		tableName pgx.Identifier,
		columnNames []string,
		rowSrc pgx.CopyFromSource,
	) (int64, error)
}

type TxHandler struct {
	pool *pgxpool.Pool
}

func NewTxHandler(pool *pgxpool.Pool) *TxHandler {
	return &TxHandler{
		pool: pool,
	}
}

const txKey = "tx"

func (t *TxHandler) Begin(ctx context.Context) (context.Context, error) {
	tx, err := t.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin tx: %w", err)
	}

	return context.WithValue(ctx, txKey, tx), nil
}

func (t *TxHandler) Commit(ctx context.Context) error {
	tx, ok := ctx.Value(txKey).(pgx.Tx)
	if !ok {
		return fmt.Errorf("tx not found in context")
	}

	err := tx.Commit(ctx)
	if err != nil {
		return fmt.Errorf("failed to commit tx: %w", err)
	}
	return nil
}

func (t *TxHandler) Rollback(ctx context.Context) error {
	tx, ok := ctx.Value(txKey).(pgx.Tx)
	if !ok {
		return fmt.Errorf("tx not found in context")
	}

	err := tx.Rollback(ctx)
	if err != nil {
		return fmt.Errorf("failed to rollback tx: %w", err)
	}
	return nil
}

func (t *TxHandler) Try(ctx context.Context) Caller {
	v, ok := ctx.Value(txKey).(pgx.Tx)
	if ok {
		return v
	}
	return t.pool
}
