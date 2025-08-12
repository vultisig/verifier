package storage

import "context"

// Tx interface to handle transactions for any db storage implementation
type Tx interface {
	Begin(ctx context.Context) (context.Context, error)
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}
