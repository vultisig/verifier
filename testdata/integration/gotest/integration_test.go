package gotest

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vultisig/verifier/config"
)

var (
	testClient  *TestClient
	fixture     *FixtureData
	plugins     []PluginConfig
	jwtToken    string
	evmFixture  *EVMFixture
	verifierURL string
)

func TestMain(m *testing.M) {
	var err error

	verifierURL = getEnv("VERIFIER_URL", "http://localhost:8080")
	jwtSecret := getEnv("JWT_SECRET", "mysecret")
	s3Endpoint := getEnv("S3_ENDPOINT", "http://localhost:9000")
	s3AccessKey := getEnv("S3_ACCESS_KEY", "minioadmin")
	s3SecretKey := getEnv("S3_SECRET_KEY", "minioadmin")
	s3Bucket := getEnv("S3_BUCKET", "vultisig-verifier")

	scriptDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get working directory: %v", err)
	}

	integrationDir := filepath.Dir(scriptDir)
	repoRoot := filepath.Dir(filepath.Dir(integrationDir))

	err = os.Chdir(repoRoot)
	if err != nil {
		log.Fatalf("Failed to change to repo root: %v", err)
	}
	log.Printf("Working directory: %s\n", repoRoot)

	fixturePath := filepath.Join(integrationDir, "fixture.json")

	log.Println("Go Integration Tests")
	log.Println("========================")
	log.Printf("Verifier URL: %s\n", verifierURL)
	log.Printf("Fixture Path: %s\n", fixturePath)
	log.Println()

	fixture, err = LoadFixture(fixturePath)
	if err != nil {
		log.Fatalf("Failed to load fixture: %v", err)
	}

	plugins = GetTestPlugins()
	log.Printf("Using %d test plugins\n", len(plugins))

	jwtToken, err = GenerateJWT(jwtSecret, fixture.Vault.PublicKey, "integration-token-1", 24)
	if err != nil {
		log.Fatalf("Failed to generate JWT: %v", err)
	}

	evmFixture, err = GenerateEVMFixture(1, "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb0", "", 21000, 0)
	if err != nil {
		log.Fatalf("Failed to generate EVM fixture: %v", err)
	}

	testClient = NewTestClient(verifierURL)

	dsn := getEnv("DATABASE_DSN", "")
	encryptionSecret := getEnv("ENCRYPTION_SECRET", "")
	if dsn == "" || encryptionSecret == "" {
		cfg, err := config.ReadVerifierConfig()
		if err != nil {
			log.Fatalf("Failed to read config: %v", err)
		}
		if dsn == "" {
			dsn = cfg.Database.DSN
		}
		if encryptionSecret == "" {
			encryptionSecret = cfg.EncryptionSecret
		}
	}
	if dsn == "" {
		log.Fatalf("DATABASE_DSN is empty. Set DATABASE_DSN env var or ensure config file has database.dsn set.")
	}
	if encryptionSecret == "" {
		log.Fatalf("ENCRYPTION_SECRET is empty. Set ENCRYPTION_SECRET env var or ensure config file has encryption_secret set.")
	}

	seeder := NewSeeder(SeederConfig{
		DSN: dsn,
		S3: S3Config{
			Endpoint:  s3Endpoint,
			Region:    "us-east-1",
			AccessKey: s3AccessKey,
			SecretKey: s3SecretKey,
			Bucket:    s3Bucket,
		},
		Fixture:          fixture,
		Plugins:          plugins,
		EncryptionSecret: encryptionSecret,
	})

	ctx := context.Background()

	log.Println("Seeding database...")
	err = seeder.SeedDatabase(ctx)
	if err != nil {
		log.Fatalf("Failed to seed database: %v", err)
	}

	log.Println("Seeding vaults to S3...")
	err = seeder.SeedVaults(ctx)
	if err != nil {
		log.Fatalf("Failed to seed vaults: %v", err)
	}

	log.Println("Waiting for verifier health...")
	err = testClient.WaitForHealth(30 * time.Second)
	if err != nil {
		log.Fatalf("Verifier not healthy: %v", err)
	}
	log.Println("   Verifier is healthy")

	log.Println("Initiating vault reshare...")
	err = initiateReshare(ctx)
	if err != nil {
		log.Fatalf("Failed to initiate reshare: %v", err)
	}

	err = waitForVault(120 * time.Second)
	if err != nil {
		log.Fatalf("Reshare timeout: %v", err)
	}
	log.Println("Vault reshares completed")

	log.Println()
	log.Println("Running tests...")
	log.Println("========================")

	code := m.Run()

	os.Exit(code)
}

func initiateReshare(ctx context.Context) error {
	for i, plugin := range plugins {
		sessionID := fmt.Sprintf("00000000-0000-0000-0001-%012d", i)

		reqBody := map[string]interface{}{
			"name":               fixture.Vault.Name,
			"public_key":         fixture.Vault.PublicKey,
			"session_id":         sessionID,
			"hex_encryption_key": fixture.Reshare.HexEncryptionKey,
			"hex_chain_code":     fixture.Reshare.HexChainCode,
			"local_party_id":     fixture.Reshare.LocalPartyID,
			"old_parties":        fixture.Reshare.OldParties,
			"email":              fixture.Reshare.Email,
			"plugin_id":          plugin.ID,
		}

		resp, err := testClient.WithJWT(jwtToken).POST("/vault/reshare", reqBody)
		if err != nil {
			return fmt.Errorf("failed to initiate reshare for %s: %w", plugin.ID, err)
		}
		resp.Body.Close()

		if resp.StatusCode != 200 {
			return fmt.Errorf("reshare request for %s returned HTTP %d", plugin.ID, resp.StatusCode)
		}
	}

	return nil
}

func waitForVault(timeout time.Duration) error {
	for _, plugin := range plugins {
		err := testClient.WithJWT(jwtToken).WaitForVault(plugin.ID, fixture.Vault.PublicKey, timeout)
		if err != nil {
			return fmt.Errorf("vault not found for %s: %w", plugin.ID, err)
		}
	}
	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getPluginSessionID(pluginIndex int) string {
	return fmt.Sprintf("00000000-0000-0000-0001-%012d", pluginIndex)
}

func getPluginAPIKey(pluginID string) string {
	return fmt.Sprintf("integration-test-apikey-%s", pluginID)
}

func getPluginPolicyID(pluginIndex int) string {
	return fmt.Sprintf("00000000-0000-0000-0000-0000000000%02d", pluginIndex+11)
}
