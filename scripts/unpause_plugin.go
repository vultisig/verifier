package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/vultisig/verifier/plugin/safety"
)

const (
	dbTimeout   = 30 * time.Second
	httpTimeout = 15 * time.Second
)

func main() {
	pluginID := flag.String("plugin", "", "Plugin ID to unpause (required)")
	reason := flag.String("reason", "", "Reason for unpausing")
	by := flag.String("by", "admin", "Who triggered the unpause")
	dbOnly := flag.Bool("db-only", false, "Only update DB, skip HTTP sync")
	syncOnly := flag.Bool("sync-only", false, "Only sync to plugin, skip DB update")
	flag.Parse()

	if *pluginID == "" {
		fmt.Fprintln(os.Stderr, "error: --plugin is required")
		flag.Usage()
		os.Exit(1)
	}

	if *dbOnly && *syncOnly {
		fmt.Fprintln(os.Stderr, "error: --db-only and --sync-only are mutually exclusive")
		os.Exit(1)
	}

	if *reason == "" {
		*reason = "manual unpause"
	}

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		fmt.Fprintln(os.Stderr, "error: DATABASE_URL environment variable is required")
		os.Exit(1)
	}

	dbCtx, dbCancel := context.WithTimeout(context.Background(), dbTimeout)
	defer dbCancel()

	pool, err := pgxpool.New(dbCtx, dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: failed to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer pool.Close()

	if !*syncOnly {
		err = updateDB(dbCtx, pool, *pluginID, *reason, *by)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: failed to update database: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Database updated successfully")
	}

	if !*dbOnly {
		serverInfo, err := getServerInfo(dbCtx, pool, *pluginID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: failed to get server info: %v\n", err)
			os.Exit(1)
		}

		httpCtx, httpCancel := context.WithTimeout(context.Background(), httpTimeout)
		defer httpCancel()

		err = syncToPlugin(httpCtx, serverInfo, *pluginID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: failed to sync to plugin: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Synced to plugin successfully")
	}

	fmt.Println("Done")
}

func updateDB(ctx context.Context, pool *pgxpool.Pool, pluginID, reason, by string) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	keysignKey := safety.KeysignFlagKey(pluginID)
	keygenKey := safety.KeygenFlagKey(pluginID)

	_, err = tx.Exec(ctx, `
		INSERT INTO control_flags (key, enabled, updated_at)
		VALUES ($1, true, NOW()), ($2, true, NOW())
		ON CONFLICT (key) DO UPDATE
		SET enabled = true, updated_at = NOW()
	`, keysignKey, keygenKey)
	if err != nil {
		return fmt.Errorf("upsert control_flags: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO plugin_pause_history (plugin_id, action, reason, triggered_by, created_at)
		VALUES ($1::plugin_id, 'manual_unpause', $2, $3, NOW())
	`, pluginID, reason, by)
	if err != nil {
		return fmt.Errorf("insert pause_history: %w", err)
	}

	err = tx.Commit(ctx)
	if err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

type serverInfo struct {
	addr   string
	apiKey string
}

func getServerInfo(ctx context.Context, pool *pgxpool.Pool, pluginID string) (*serverInfo, error) {
	var addr string
	err := pool.QueryRow(ctx, `
		SELECT server_endpoint FROM plugins WHERE id = $1::plugin_id
	`, pluginID).Scan(&addr)
	if err != nil {
		return nil, fmt.Errorf("get server_endpoint: %w", err)
	}

	var apiKey string
	err = pool.QueryRow(ctx, `
		SELECT apikey FROM plugin_apikey
		WHERE plugin_id = $1::plugin_id AND status = 1
		ORDER BY created_at DESC
		LIMIT 1
	`, pluginID).Scan(&apiKey)
	if err != nil {
		return nil, fmt.Errorf("get apikey: %w", err)
	}

	return &serverInfo{addr: strings.TrimRight(addr, "/"), apiKey: apiKey}, nil
}

func syncToPlugin(ctx context.Context, info *serverInfo, pluginID string) error {
	flags := []safety.ControlFlag{
		{Key: safety.KeysignFlagKey(pluginID), Enabled: true},
		{Key: safety.KeygenFlagKey(pluginID), Enabled: true},
	}

	body, err := json.Marshal(flags)
	if err != nil {
		return fmt.Errorf("marshal flags: %w", err)
	}

	url := info.addr + "/plugin/safety"
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+info.apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("http request: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}
