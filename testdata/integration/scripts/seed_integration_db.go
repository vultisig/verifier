package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vultisig/verifier/config"
	"gopkg.in/yaml.v3"
)

type ProposedYAML struct {
	Plugins []PluginConfig `yaml:"plugins"`
}

type PluginConfig struct {
	ID             string   `yaml:"id"`
	Title          string   `yaml:"title"`
	Description    string   `yaml:"description"`
	ServerEndpoint string   `yaml:"server_endpoint"`
	Category       string   `yaml:"category"`
	LogoURL        string   `yaml:"logo_url"`
	ThumbnailURL   string   `yaml:"thumbnail_url"`
	Audited        bool     `yaml:"audited"`
	Features       []string `yaml:"features"`
	FAQs           []struct {
		Question string `yaml:"question"`
		Answer   string `yaml:"answer"`
	} `yaml:"faqs"`
}

func main() {
	ctx := context.Background()

	cfg, err := config.ReadVerifierConfig()
	if err != nil {
		panic(err)
	}

	// Read proposed.yaml
	proposedYAML, err := os.ReadFile("proposed.yaml")
	if err != nil {
		log.Fatalf("Failed to read proposed.yaml: %v", err)
	}

	var proposed ProposedYAML
	if err := yaml.Unmarshal(proposedYAML, &proposed); err != nil {
		log.Fatalf("Failed to parse proposed.yaml: %v", err)
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

	fmt.Println("🌱 Seeding integration database with plugins from proposed.yaml...")

	// Insert plugins from proposed.yaml
	for _, plugin := range proposed.Plugins {
		fmt.Printf("  📦 Inserting plugin: %s...\n", plugin.ID)

		_, err := tx.Exec(ctx, `
			INSERT INTO plugins (id, title, description, server_endpoint, category, logo_url, thumbnail_url, audited)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			ON CONFLICT (id) DO UPDATE SET
				title = EXCLUDED.title,
				description = EXCLUDED.description,
				server_endpoint = EXCLUDED.server_endpoint,
				category = EXCLUDED.category,
				logo_url = EXCLUDED.logo_url,
				thumbnail_url = EXCLUDED.thumbnail_url,
				audited = EXCLUDED.audited,
				updated_at = NOW()
		`, plugin.ID, plugin.Title, plugin.Description, plugin.ServerEndpoint,
			plugin.Category, plugin.LogoURL, plugin.ThumbnailURL, plugin.Audited)

		if err != nil {
			tx.Rollback(ctx)
			log.Fatalf("Failed to insert plugin %s: %v", plugin.ID, err)
		}

		// Insert API key for the plugin (required for integration tests)
		apiKey := fmt.Sprintf("integration-test-apikey-%s", plugin.ID)
		_, err = tx.Exec(ctx, `
			INSERT INTO plugin_apikey (id, plugin_id, apikey, created_at, expires_at, status)
			VALUES (gen_random_uuid(), $1, $2, NOW(), NULL, 1)
			ON CONFLICT DO NOTHING
		`, plugin.ID, apiKey)

		if err != nil {
			tx.Rollback(ctx)
			log.Fatalf("Failed to insert API key for plugin %s: %v", plugin.ID, err)
		}

		fmt.Printf("  ✅ Plugin %s seeded successfully (API Key: %s)\n", plugin.ID, apiKey)
	}

	// If we got here, everything succeeded - commit the transaction
	if err := tx.Commit(ctx); err != nil {
		log.Fatalf("Failed to commit transaction: %v", err)
	}

	fmt.Println("✅ Integration database seeding completed successfully!")
	fmt.Printf("   Total plugins seeded: %d\n", len(proposed.Plugins))
}
