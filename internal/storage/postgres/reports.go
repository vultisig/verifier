package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	itypes "github.com/vultisig/verifier/internal/types"
	"github.com/vultisig/verifier/types"
)

const (
	PLUGIN_REPORTS_TABLE       = "plugin_reports"
	PLUGIN_PAUSE_HISTORY_TABLE = "plugin_pause_history"
)

func (p *PostgresBackend) UpsertReport(ctx context.Context, pluginID types.PluginID, publicKey, reason string, cooldown time.Duration) error {
	if p.pool == nil {
		return fmt.Errorf("database pool is nil")
	}

	intervalStr := fmt.Sprintf("%d seconds", int64(cooldown.Seconds()))

	query := fmt.Sprintf(`
		INSERT INTO %s (plugin_id, reporter_public_key, reason, created_at, last_reported_at, report_count)
		VALUES ($1, $2, $3, NOW(), NOW(), 1)
		ON CONFLICT (plugin_id, reporter_public_key) DO UPDATE
		SET last_reported_at = NOW(),
		    reason = EXCLUDED.reason,
		    report_count = %s.report_count + 1
		WHERE %s.last_reported_at < NOW() - $4::interval`,
		PLUGIN_REPORTS_TABLE, PLUGIN_REPORTS_TABLE, PLUGIN_REPORTS_TABLE)

	_, err := p.pool.Exec(ctx, query, pluginID, publicKey, reason, intervalStr)
	if err != nil {
		return fmt.Errorf("failed to upsert report: %w", err)
	}

	return nil
}

func (p *PostgresBackend) GetReport(ctx context.Context, pluginID types.PluginID, publicKey string) (*itypes.PluginReport, error) {
	if p.pool == nil {
		return nil, fmt.Errorf("database pool is nil")
	}

	query := fmt.Sprintf(`
		SELECT plugin_id, reporter_public_key, reason, created_at, last_reported_at, report_count
		FROM %s
		WHERE plugin_id = $1 AND reporter_public_key = $2`,
		PLUGIN_REPORTS_TABLE)

	var report itypes.PluginReport
	err := p.pool.QueryRow(ctx, query, pluginID, publicKey).Scan(
		&report.PluginID,
		&report.ReporterPubKey,
		&report.Reason,
		&report.CreatedAt,
		&report.LastReportedAt,
		&report.ReportCount,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get report: %w", err)
	}

	return &report, nil
}

func (p *PostgresBackend) CountReportsInWindow(ctx context.Context, pluginID types.PluginID, window time.Duration) (int, error) {
	if p.pool == nil {
		return 0, fmt.Errorf("database pool is nil")
	}

	intervalStr := fmt.Sprintf("%d seconds", int64(window.Seconds()))

	query := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM %s
		WHERE plugin_id = $1 AND last_reported_at >= NOW() - $2::interval`,
		PLUGIN_REPORTS_TABLE)

	var count int
	err := p.pool.QueryRow(ctx, query, pluginID, intervalStr).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count reports: %w", err)
	}

	return count, nil
}

func (p *PostgresBackend) HasInstallation(ctx context.Context, pluginID types.PluginID, publicKey string) (bool, error) {
	if p.pool == nil {
		return false, fmt.Errorf("database pool is nil")
	}

	query := fmt.Sprintf(`
		SELECT EXISTS(
			SELECT 1 FROM %s
			WHERE plugin_id = $1 AND public_key = $2
		)`, PLUGIN_INSTALLATIONS_TABLE)

	var exists bool
	err := p.pool.QueryRow(ctx, query, pluginID, publicKey).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check installation: %w", err)
	}

	return exists, nil
}

func (p *PostgresBackend) CountInstallations(ctx context.Context, pluginID types.PluginID) (int, error) {
	if p.pool == nil {
		return 0, fmt.Errorf("database pool is nil")
	}

	query := fmt.Sprintf(`
		SELECT COUNT(*)
		FROM %s
		WHERE plugin_id = $1`, PLUGIN_INSTALLATIONS_TABLE)

	var count int
	err := p.pool.QueryRow(ctx, query, pluginID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count installations: %w", err)
	}

	return count, nil
}


func (p *PostgresBackend) IsPluginPaused(ctx context.Context, pluginID types.PluginID) (bool, error) {
	if p.pool == nil {
		return false, fmt.Errorf("database pool is nil")
	}

	keysignKey := string(pluginID) + "-keysign"

	query := `
		SELECT enabled
		FROM control_flags
		WHERE key = $1`

	var enabled bool
	err := p.pool.QueryRow(ctx, query, keysignKey).Scan(&enabled)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check plugin pause status: %w", err)
	}

	return !enabled, nil
}

func (p *PostgresBackend) setControlFlagTx(ctx context.Context, tx pgx.Tx, key string, enabled bool) error {
	query := `
		INSERT INTO control_flags (key, enabled, updated_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (key) DO UPDATE
		SET enabled = EXCLUDED.enabled, updated_at = NOW()`

	_, err := tx.Exec(ctx, query, key, enabled)
	if err != nil {
		return fmt.Errorf("failed to set control flag: %w", err)
	}

	return nil
}

func (p *PostgresBackend) recordPauseHistoryTx(ctx context.Context, tx pgx.Tx, record itypes.PauseHistoryRecord) error {
	query := fmt.Sprintf(`
		INSERT INTO %s (plugin_id, action, report_count_window, active_users, threshold_rate, reason, triggered_by, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())`,
		PLUGIN_PAUSE_HISTORY_TABLE)

	_, err := tx.Exec(ctx, query,
		record.PluginID,
		record.Action,
		record.ReportCountWindow,
		record.ActiveUsers,
		record.ThresholdRate,
		record.Reason,
		record.TriggeredBy,
	)
	if err != nil {
		return fmt.Errorf("failed to record pause history: %w", err)
	}

	return nil
}

func (p *PostgresBackend) PausePlugin(ctx context.Context, pluginID types.PluginID, record itypes.PauseHistoryRecord) error {
	return p.WithTransaction(ctx, func(ctx context.Context, tx pgx.Tx) error {
		keysignKey := string(pluginID) + "-keysign"
		keygenKey := string(pluginID) + "-keygen"

		err := p.setControlFlagTx(ctx, tx, keysignKey, false)
		if err != nil {
			return fmt.Errorf("failed to set keysign flag: %w", err)
		}

		err = p.setControlFlagTx(ctx, tx, keygenKey, false)
		if err != nil {
			return fmt.Errorf("failed to set keygen flag: %w", err)
		}

		err = p.recordPauseHistoryTx(ctx, tx, record)
		if err != nil {
			return fmt.Errorf("failed to record pause history: %w", err)
		}

		return nil
	})
}

