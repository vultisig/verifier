package main

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vultisig/verifier/config"
)

func main() {
	ctx := context.Background()

	cfg, err := config.ReadVerifierConfig()
	if err != nil {
		panic(err)
	}

	pool, err := pgxpool.New(ctx, cfg.Database.DSN)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	// Start a transaction - if anything fails, everything rolls back
	tx, err := pool.Begin(ctx)
	if err != nil {
		log.Fatalf("Failed to begin transaction: %v", err)
	}

	// Ensure transaction cleanup
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback(ctx)
			panic(r)
		}
	}()

	// Find SQL files
	sqlFiles, err := getSQLFiles("testdata/seeds")
	if err != nil {
		tx.Rollback(ctx)
		log.Fatalf("Failed to find SQL files: %v", err)
	}

	if len(sqlFiles) == 0 {
		tx.Rollback(ctx)
		log.Fatalf("No SQL files found in testdata/seeds")
	}

	fmt.Println("🌱 Seeding database (with transaction protection)...")

	// Run each SQL file within the transaction
	for _, file := range sqlFiles {
		fmt.Printf("  📄 Running %s...\n", filepath.Base(file))

		sqlContent, err := os.ReadFile(file)
		if err != nil {
			tx.Rollback(ctx)
			log.Fatalf("Failed to read %s: %v", file, err)
		}

		if _, err := tx.Exec(ctx, string(sqlContent)); err != nil {
			tx.Rollback(ctx)
			log.Fatalf("Failed to execute %s: %v", file, err)
		}
	}

	// If we got here, everything succeeded - commit the transaction
	if err := tx.Commit(ctx); err != nil {
		log.Fatalf("Failed to commit transaction: %v", err)
	}

	fmt.Println("✅ Database seeding completed successfully!")
}

func getSQLFiles(dir string) ([]string, error) {
	var sqlFiles []string

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(strings.ToLower(path), ".sql") {
			sqlFiles = append(sqlFiles, path)
		}
		return nil
	})

	sort.Strings(sqlFiles)
	return sqlFiles, err
}
