package postgres

import (
	"embed"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/sirupsen/logrus"
)

//go:embed migrations/plugin/*.sql
var pluginMigrations embed.FS

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
