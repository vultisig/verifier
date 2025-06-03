package storage

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RowsStream[T any] struct {
	Row T
	Err error
}

// GetRowsStream
// TLDR: fetch rows from db with a non-buffered channel to control concurrency by data-consumer
func GetRowsStream[T any](
	ctx context.Context,
	pool *pgxpool.Pool,
	scanRow func(rows pgx.Rows) (T, error),
	sql string,
	args ...any,
) <-chan RowsStream[T] {
	ch := make(chan RowsStream[T])

	go func() {
		defer close(ch)

		rows, err := pool.Query(
			ctx,
			sql,
			args...,
		)
		if err != nil {
			ch <- RowsStream[T]{Err: fmt.Errorf("p.pool.Query: %w", err)}
			return
		}

		for rows.Next() {
			item, er := scanRow(rows)
			if er != nil {
				ch <- RowsStream[T]{Err: fmt.Errorf("scanRow: %w", er)}
				return
			}

			ch <- RowsStream[T]{Row: item}
		}
	}()

	return ch
}
