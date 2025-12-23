package main

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vultisig/verifier/config"
)

func main() {
	ctx := context.Background()

	cfg, err := config.ReadVerifierConfig()
	if err != nil {
		panic(err)
	}

	err = seedS3(cfg)
	if err != nil {
		log.Fatalf("seedS3: %v", err)
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

func seedS3(cfg *config.VerifierConfig) error {
	fmt.Println("🌱 Seeding S3...")

	// better don't use vault.NewBlockStorageImp(cfg.BlockStorage) here,
	// because it has DKLS as dependency and it will try to load it locally
	sess, err := session.NewSession(&aws.Config{
		Region:           aws.String(cfg.BlockStorage.Region),
		Endpoint:         aws.String(cfg.BlockStorage.Host),
		Credentials:      credentials.NewStaticCredentials(cfg.BlockStorage.AccessKey, cfg.BlockStorage.SecretKey, ""),
		S3ForcePathStyle: aws.Bool(true),
	})
	s3Client := s3.New(sess)

	keysharesDir := path.Join("testdata", "keyshares")
	keyshares, err := os.ReadDir(keysharesDir)
	if err != nil {
		return fmt.Errorf("os.ReadDir('%s'): %w", keysharesDir, err)
	}

	for _, file := range keyshares {
		if file.IsDir() {
			continue
		}

		filePath := path.Join(keysharesDir, file.Name())
		f, er := os.Open(filePath)
		if er != nil {
			panic(fmt.Errorf("os.ReadFile('%s'): %w", filePath, er))
		}

		fmt.Printf("  📄 Uploading %s...\n", filePath)

		_, er = s3Client.PutObject(&s3.PutObjectInput{
			Bucket: aws.String(cfg.BlockStorage.Bucket),
			Key:    aws.String(file.Name()),
			Body:   f,
		})
		_ = f.Close()
		if er != nil {
			return fmt.Errorf("s3.SaveVault('%s'): %w", file.Name(), er)
		}
	}

	fmt.Println("✅ S3 seeding completed successfully!")
	return nil
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
