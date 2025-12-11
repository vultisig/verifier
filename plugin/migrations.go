package plugin

import (
	"embed"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/sirupsen/logrus"
)

//go:embed scheduler/scheduler_pg/migrations/*.sql
//go:embed policy/policy_pg/migrations/*.sql
//go:embed tx_indexer/pkg/storage/migrations/*.sql
var pluginMigrations embed.FS

func WithMigrations[T any](
	logger *logrus.Logger,
	pool *pgxpool.Pool,
	constructor func(*pgxpool.Pool) T,
	migrationsDir string,
) (T, error) {
	migrationManager := NewMigrationManager(logger, pool, migrationsDir)
	err := migrationManager.Migrate()
	if err != nil {
		return *new(T), fmt.Errorf("failed to run migrations: %w", err)
	}

	return constructor(pool), nil
}

// MigrationManager handles plugin-specific migrations
type MigrationManager struct {
	logger *logrus.Logger
	pool   *pgxpool.Pool
	dir    string
}

func NewMigrationManager(logger *logrus.Logger, pool *pgxpool.Pool, dir string) *MigrationManager {
	return &MigrationManager{
		logger: logger.WithField("pkg", "postgres.MigrationManager").Logger,
		pool:   pool,
		dir:    dir,
	}
}

func (m *MigrationManager) Migrate() error {
	m.logger.Info("Starting plugin database migration...")
	goose.SetLogger(logrus.StandardLogger())
	goose.SetBaseFS(pluginMigrations)
	defer goose.SetBaseFS(nil)
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("failed to set goose dialect: %w", err)
	}

	db := stdlib.OpenDBFromPool(m.pool)
	defer func() {
		_ = db.Close()
	}()
	if err := goose.Up(db, m.dir, goose.WithAllowMissing()); err != nil {
		return fmt.Errorf("failed to run plugin migrations: %w", err)
	}
	m.logger.Info("Plugin database migration completed successfully")
	return nil
}
