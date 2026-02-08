package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/vultisig/verifier/internal/safety"
	itypes "github.com/vultisig/verifier/internal/types"
	psafety "github.com/vultisig/verifier/plugin/safety"
)

var (
	ErrNotEligible    = errors.New("not eligible to report: no installation found")
	ErrCooldownActive = errors.New("cooldown active")
)

type ReportServiceStorage interface {
	UpsertReport(ctx context.Context, pluginID string, publicKey, reason, details string, cooldown time.Duration) error
	GetReport(ctx context.Context, pluginID string, publicKey string) (*itypes.PluginReport, error)
	CountReportsInWindow(ctx context.Context, pluginID string, window time.Duration) (int, error)
	HasInstallation(ctx context.Context, pluginID string, publicKey string) (bool, error)
	CountInstallations(ctx context.Context, pluginID string) (int, error)
	IsPluginPaused(ctx context.Context, pluginID string) (bool, error)
	PausePlugin(ctx context.Context, pluginID string, record itypes.PauseHistoryRecord) error
}

type SafetySyncer interface {
	SyncSafetyToPlugin(ctx context.Context, pluginID string, flags []psafety.ControlFlag) error
}

type ReportService struct {
	db     ReportServiceStorage
	syncer SafetySyncer
	logger *logrus.Logger
}

func NewReportService(db ReportServiceStorage, syncer SafetySyncer, logger *logrus.Logger) (*ReportService, error) {
	if db == nil {
		return nil, fmt.Errorf("database storage cannot be nil")
	}
	if logger == nil {
		return nil, fmt.Errorf("logger cannot be nil")
	}
	return &ReportService{
		db:     db,
		syncer: syncer,
		logger: logger,
	}, nil
}

func (s *ReportService) SubmitReport(ctx context.Context, pluginID string, publicKey, reason, details string) (*itypes.ReportSubmitResult, error) {
	hasInstallation, err := s.db.HasInstallation(ctx, pluginID, publicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to check installation: %w", err)
	}
	if !hasInstallation {
		return nil, ErrNotEligible
	}

	isPaused, err := s.db.IsPluginPaused(ctx, pluginID)
	if err != nil {
		return nil, fmt.Errorf("failed to check pause status: %w", err)
	}

	cooldown := safety.ReportCooldown
	if isPaused {
		cooldown = safety.ReportPausedCooldown
	}

	existingReport, err := s.db.GetReport(ctx, pluginID, publicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing report: %w", err)
	}

	if existingReport != nil {
		timeSinceLastReport := time.Since(existingReport.LastReportedAt)
		if timeSinceLastReport < cooldown {
			remaining := cooldown - timeSinceLastReport
			return nil, fmt.Errorf("%w: please wait %s before reporting again", ErrCooldownActive, remaining.Round(time.Minute))
		}
	}

	err = s.db.UpsertReport(ctx, pluginID, publicKey, reason, details, cooldown)
	if err != nil {
		if errors.Is(err, itypes.ErrReportCooldown) {
			return nil, fmt.Errorf("%w: concurrent request detected", ErrCooldownActive)
		}
		return nil, fmt.Errorf("failed to upsert report: %w", err)
	}

	if !isPaused {
		err = s.evaluateAndPause(ctx, pluginID)
		if err != nil {
			s.logger.WithError(err).WithField("plugin_id", pluginID).Error("failed to evaluate auto-pause")
		}
		isPaused, err = s.db.IsPluginPaused(ctx, pluginID)
		if err != nil {
			s.logger.WithError(err).WithField("plugin_id", pluginID).Error("failed to check pause status after evaluation")
		}
	}

	return &itypes.ReportSubmitResult{
		Status:       "recorded",
		PluginPaused: isPaused,
	}, nil
}

func (s *ReportService) evaluateAndPause(ctx context.Context, pluginID string) error {
	isPaused, err := s.db.IsPluginPaused(ctx, pluginID)
	if err != nil {
		return fmt.Errorf("failed to check pause status: %w", err)
	}
	if isPaused {
		return nil
	}

	reportsInWindow, err := s.db.CountReportsInWindow(ctx, pluginID, safety.ReportsWindowDuration)
	if err != nil {
		return fmt.Errorf("failed to count reports: %w", err)
	}

	activeUsers, err := s.db.CountInstallations(ctx, pluginID)
	if err != nil {
		return fmt.Errorf("failed to count installations: %w", err)
	}

	shouldPause, rate, threshold := safety.ShouldAutoPause(reportsInWindow, activeUsers)
	if !shouldPause {
		return nil
	}

	s.logger.WithFields(logrus.Fields{
		"plugin_id":         pluginID,
		"reports_in_window": reportsInWindow,
		"active_users":      activeUsers,
		"rate":              rate,
		"threshold":         threshold,
	}).Warn("auto-pausing plugin due to reports")

	reportCount := reportsInWindow
	users := activeUsers
	thresholdRate := threshold
	reason := fmt.Sprintf("%d reports (%.1f%%) exceeded threshold (%.1f%%)", reportsInWindow, rate*100, threshold*100)
	triggeredBy := "system"

	err = s.db.PausePlugin(ctx, pluginID, itypes.PauseHistoryRecord{
		PluginID:          pluginID,
		Action:            "auto_paused",
		ReportCountWindow: &reportCount,
		ActiveUsers:       &users,
		ThresholdRate:     &thresholdRate,
		Reason:            &reason,
		TriggeredBy:       &triggeredBy,
	})
	if err != nil {
		return fmt.Errorf("failed to pause plugin: %w", err)
	}

	if s.syncer != nil {
		flags := []psafety.ControlFlag{
			{Key: psafety.KeysignFlagKey(string(pluginID)), Enabled: false},
			{Key: psafety.KeygenFlagKey(string(pluginID)), Enabled: false},
		}
		syncErr := s.syncer.SyncSafetyToPlugin(ctx, pluginID, flags)
		if syncErr != nil {
			s.logger.WithError(syncErr).WithField("plugin_id", pluginID).Warn("failed to sync safety to plugin")
		}
	}

	return nil
}
