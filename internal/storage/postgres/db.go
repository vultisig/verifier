package postgres

import (
	"context"
	"embed"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/sirupsen/logrus"

	"github.com/vultisig/verifier/internal/storage"
)

//go:embed migrations/*
var embeddedMigrations embed.FS

var _ storage.DatabaseStorage = (*PostgresBackend)(nil)

type PostgresBackend struct {
	pool *pgxpool.Pool
}

const defaultTimeout = 10 * time.Second

func NewPostgresBackend(dsn string) (*PostgresBackend, error) {
	logrus.Info("Connecting to database with DSN: ", dsn)
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	backend := &PostgresBackend{
		pool: pool,
	}

	if err := backend.Migrate(); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return backend, nil
}

func (p *PostgresBackend) Close() error {
	p.pool.Close()

	return nil
}

func (p *PostgresBackend) Migrate() error {
	logrus.Info("Starting database migration...")
	goose.SetBaseFS(embeddedMigrations)
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("failed to set goose dialect: %w", err)
	}

	db := stdlib.OpenDBFromPool(p.pool)
	if err := goose.Up(db, "migrations", goose.WithAllowMissing()); err != nil {
		return fmt.Errorf("failed to run goose up: %w", err)
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
