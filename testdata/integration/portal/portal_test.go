package portal

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vultisig/verifier/internal/portal"
	"github.com/vultisig/verifier/internal/types"
)

var (
	portalURL       string
	portalJWTSecret string
	testPluginID    string
	pool            *pgxpool.Pool
	authService     *portal.PortalAuthService
)

func TestMain(m *testing.M) {
	portalURL = getEnv("PORTAL_URL", "http://localhost:8081")
	portalJWTSecret = getEnv("PORTAL_JWT_SECRET", "test-portal-secret")
	testPluginID = getEnv("TEST_PLUGIN_ID", "vultisig-dca-0000")

	dsn := getEnv("DATABASE_DSN", "")
	if dsn == "" {
		log.Println("DATABASE_DSN not set, skipping portal integration tests")
		os.Exit(0)
	}

	var err error
	pool, err = pgxpool.New(context.Background(), dsn)
	if err != nil {
		log.Printf("Failed to connect to database: %v", err)
		os.Exit(1)
	}

	authService = portal.NewPortalAuthService(portalJWTSecret, logrus.New())

	log.Println("Portal Integration Tests")
	log.Println("========================")
	log.Printf("Portal URL: %s", portalURL)
	log.Printf("Test Plugin ID: %s", testPluginID)

	if err := ensureTestPlugin(context.Background()); err != nil {
		log.Printf("Failed to ensure test plugin exists: %v", err)
		pool.Close()
		os.Exit(1)
	}

	if err := waitForPortalHealth(10 * time.Second); err != nil {
		log.Printf("Portal not healthy: %v", err)
		pool.Close()
		os.Exit(1)
	}
	log.Println("Portal is healthy")

	code := m.Run()
	pool.Close()
	os.Exit(code)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func ensureTestPlugin(ctx context.Context) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO plugins (id, title, description, server_endpoint, category, audited)
		VALUES ($1, 'Test Plugin', 'Test plugin for integration tests', 'http://localhost:9999', 'plugin', false)
		ON CONFLICT (id) DO NOTHING
	`, testPluginID)
	return err
}

func waitForPortalHealth(timeout time.Duration) error {
	client := &http.Client{Timeout: 5 * time.Second}
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		resp, err := client.Get(portalURL + "/healthz")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	return fmt.Errorf("portal not healthy after %v", timeout)
}

func seedPluginOwner(ctx context.Context, pluginID, address string) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO plugin_owners (plugin_id, public_key, role, added_via, active)
		VALUES ($1, $2, $3, $4, true)
		ON CONFLICT (plugin_id, public_key) DO UPDATE SET
			role = $3,
			active = true
	`, pluginID, address, types.PluginOwnerRoleAdmin, types.PluginOwnerAddedViaAdminCLI)
	return err
}

func cleanupApiKeys(ctx context.Context, pluginID string) error {
	_, err := pool.Exec(ctx, `DELETE FROM plugin_apikey WHERE plugin_id = $1`, pluginID)
	return err
}

func countApiKeys(ctx context.Context, pluginID string) (int64, error) {
	var count int64
	err := pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM plugin_apikey
		WHERE plugin_id = $1 AND status = 1 AND (expires_at IS NULL OR expires_at > NOW())
	`, pluginID).Scan(&count)
	return count, err
}

func TestApiKeyLimit(t *testing.T) {
	ctx := context.Background()
	testAddress := "0x1234567890123456789012345678901234567890"

	err := seedPluginOwner(ctx, testPluginID, testAddress)
	require.NoError(t, err, "Failed to seed plugin owner")

	err = cleanupApiKeys(ctx, testPluginID)
	require.NoError(t, err, "Failed to cleanup existing API keys")

	token, err := authService.GenerateToken("test-pubkey", testAddress)
	require.NoError(t, err, "Failed to generate JWT")

	client := &http.Client{Timeout: 10 * time.Second}

	createApiKey := func() (*http.Response, error) {
		url := fmt.Sprintf("%s/plugins/%s/api-keys", portalURL, testPluginID)
		req, err := http.NewRequest(http.MethodPost, url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		return client.Do(req)
	}

	t.Run("create keys up to limit", func(t *testing.T) {
		maxKeys := 5

		for i := 0; i < maxKeys; i++ {
			resp, err := createApiKey()
			require.NoError(t, err)
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()

			assert.Equal(t, http.StatusCreated, resp.StatusCode,
				"Expected 201 for key %d, got %d: %s", i+1, resp.StatusCode, string(body))
		}

		count, err := countApiKeys(ctx, testPluginID)
		require.NoError(t, err)
		assert.Equal(t, int64(maxKeys), count)
	})

	t.Run("reject when limit reached", func(t *testing.T) {
		resp, err := createApiKey()
		require.NoError(t, err)
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		assert.Equal(t, http.StatusConflict, resp.StatusCode,
			"Expected 409 Conflict when limit reached, got %d: %s", resp.StatusCode, string(body))

		var errResp map[string]string
		err = json.Unmarshal(body, &errResp)
		require.NoError(t, err)
		assert.Contains(t, errResp["error"], "maximum number of API keys")
	})

	t.Run("cleanup", func(t *testing.T) {
		err := cleanupApiKeys(ctx, testPluginID)
		require.NoError(t, err)
	})
}

func TestApiKeyLimitAfterExpiry(t *testing.T) {
	ctx := context.Background()
	testAddress := "0x1234567890123456789012345678901234567891"

	err := seedPluginOwner(ctx, testPluginID, testAddress)
	require.NoError(t, err)

	err = cleanupApiKeys(ctx, testPluginID)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `
		INSERT INTO plugin_apikey (plugin_id, apikey, status, expires_at)
		VALUES ($1, 'expired-key-1', 1, NOW() - INTERVAL '1 hour'),
		       ($1, 'expired-key-2', 1, NOW() - INTERVAL '1 hour'),
		       ($1, 'expired-key-3', 1, NOW() - INTERVAL '1 hour'),
		       ($1, 'expired-key-4', 1, NOW() - INTERVAL '1 hour'),
		       ($1, 'expired-key-5', 1, NOW() - INTERVAL '1 hour')
	`, testPluginID)
	require.NoError(t, err)

	count, err := countApiKeys(ctx, testPluginID)
	require.NoError(t, err)
	assert.Equal(t, int64(0), count, "Expired keys should not count")

	token, err := authService.GenerateToken("test-pubkey", testAddress)
	require.NoError(t, err)

	client := &http.Client{Timeout: 10 * time.Second}
	url := fmt.Sprintf("%s/plugins/%s/api-keys", portalURL, testPluginID)
	req, err := http.NewRequest(http.MethodPost, url, nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	require.NoError(t, err)
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	assert.Equal(t, http.StatusCreated, resp.StatusCode,
		"Should allow new key when existing keys are expired: %s", string(body))

	err = cleanupApiKeys(ctx, testPluginID)
	require.NoError(t, err)
}

func TestApiKeyLimitConcurrent(t *testing.T) {
	ctx := context.Background()
	testAddress := "0x1234567890123456789012345678901234567892"

	err := seedPluginOwner(ctx, testPluginID, testAddress)
	require.NoError(t, err, "Failed to seed plugin owner")

	err = cleanupApiKeys(ctx, testPluginID)
	require.NoError(t, err, "Failed to cleanup existing API keys")

	token, err := authService.GenerateToken("test-pubkey", testAddress)
	require.NoError(t, err, "Failed to generate JWT")

	client := &http.Client{Timeout: 30 * time.Second}

	createApiKey := func() int {
		url := fmt.Sprintf("%s/plugins/%s/api-keys", portalURL, testPluginID)
		req, err := http.NewRequest(http.MethodPost, url, nil)
		if err != nil {
			return 0
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			return 0
		}
		defer resp.Body.Close()
		io.ReadAll(resp.Body)
		return resp.StatusCode
	}

	numRequests := 10
	maxKeys := 5

	var wg sync.WaitGroup
	results := make(chan int, numRequests)

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results <- createApiKey()
		}()
	}

	wg.Wait()
	close(results)

	created, conflict := 0, 0
	for code := range results {
		if code == http.StatusCreated {
			created++
		}
		if code == http.StatusConflict {
			conflict++
		}
	}

	assert.Equal(t, maxKeys, created, "Should create exactly %d keys", maxKeys)
	assert.Equal(t, numRequests-maxKeys, conflict, "Should reject %d keys", numRequests-maxKeys)

	count, err := countApiKeys(ctx, testPluginID)
	require.NoError(t, err)
	assert.Equal(t, int64(maxKeys), count, "Database should have exactly %d active keys", maxKeys)

	err = cleanupApiKeys(ctx, testPluginID)
	require.NoError(t, err)
}
