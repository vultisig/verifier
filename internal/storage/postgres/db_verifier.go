package postgres

import (
	"embed"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/sirupsen/logrus"
)

//go:embed migrations/verifier/*.sql
var verifierMigrations embed.FS

// VerifierMigrationManager handles verifier-specific migrations
type VerifierMigrationManager struct {
	pool *pgxpool.Pool
}

func NewVerifierMigrationManager(pool *pgxpool.Pool) *VerifierMigrationManager {
	return &VerifierMigrationManager{pool: pool}
}

func (v *VerifierMigrationManager) Migrate() error {
	logrus.Info("Starting verifier database migration...")
	goose.SetBaseFS(verifierMigrations)
	defer goose.SetBaseFS(nil)
	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("failed to set goose dialect: %w", err)
	}

	db := stdlib.OpenDBFromPool(v.pool)
	defer db.Close()
	if err := goose.Up(db, "migrations/verifier", goose.WithAllowMissing()); err != nil {
		return fmt.Errorf("failed to run verifier migrations: %w", err)
	}
	logrus.Info("Verifier database migration completed successfully")
	return nil
}
