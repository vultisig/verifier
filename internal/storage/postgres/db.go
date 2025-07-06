package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sirupsen/logrus"

	"github.com/vultisig/verifier/internal/storage"
)

var _ storage.DatabaseStorage = (*PostgresBackend)(nil)

type PostgresBackend struct {
	pool *pgxpool.Pool
}

type MigrationOptions struct {
	RunSystemMigrations   bool
	RunVerifierMigrations bool
}

func NewPostgresBackend(dsn string, opts *MigrationOptions) (*PostgresBackend, error) {
	logrus.Info("Connecting to database with DSN: ", dsn)
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	backend := &PostgresBackend{
		pool: pool,
	}

	// Apply default options if not provided
	if opts == nil {
		opts = &MigrationOptions{
			RunSystemMigrations:   true,
			RunVerifierMigrations: true,
		}
	}

	if err := backend.Migrate(opts); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return backend, nil
}

func (p *PostgresBackend) Close() error {
	p.pool.Close()

	return nil
}

func (p *PostgresBackend) Migrate(opts *MigrationOptions) error {
	logrus.Info("Starting database migration...")

	// Run system migrations first (plugin_policies table)
	if opts.RunSystemMigrations {
		systemMgr := NewSystemMigrationManager(p.pool)
		if err := systemMgr.Migrate(); err != nil {
			return fmt.Errorf("failed to run system migrations: %w", err)
		}
	}

	// Run verifier migrations (all other tables)
	if opts.RunVerifierMigrations {
		verifierMgr := NewVerifierMigrationManager(p.pool)
		if err := verifierMgr.Migrate(); err != nil {
			return fmt.Errorf("failed to run verifier migrations: %w", err)
		}
	}

	logrus.Info("Database migration completed successfully")
	return nil
}

func (p *PostgresBackend) WithTransaction(ctx context.Context, fn func(ctx context.Context, tx pgx.Tx) error) error {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Roll back on error *or* panic.
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx)
			panic(p)
		}
	}()

	if err := fn(ctx, tx); err != nil {
		_ = tx.Rollback(ctx)
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func (p *PostgresBackend) Pool() *pgxpool.Pool {
	return p.pool
}
