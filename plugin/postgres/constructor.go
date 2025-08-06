package postgres

import (
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sirupsen/logrus"
)

type StorageConstructor[T any] func(*pgxpool.Pool) T

func WithMigrations[T any](
	logger *logrus.Logger,
	pool *pgxpool.Pool,
	constructor StorageConstructor[T],
	migrationsDir string,
) (T, error) {
	migrationManager := NewMigrationManager(logger, pool, migrationsDir)
	err := migrationManager.Migrate()
	if err != nil {
		return *new(T), fmt.Errorf("failed to run migrations: %w", err)
	}

	return constructor(pool), nil
}
