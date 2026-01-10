package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/vultisig/verifier/internal/safety"
	itypes "github.com/vultisig/verifier/internal/types"
	"github.com/vultisig/verifier/types"
)

var (
	ErrNotEligible = errors.New("not eligible to report: no installation found")
)

type ReportServiceStorage interface {
	UpsertReport(ctx context.Context, pluginID types.PluginID, publicKey, reason string, cooldown time.Duration) error
	GetReport(ctx context.Context, pluginID types.PluginID, publicKey string) (*itypes.PluginReport, error)
	CountReportsInWindow(ctx context.Context, pluginID types.PluginID, window time.Duration) (int, error)
	HasInstallation(ctx context.Context, pluginID types.PluginID, publicKey string) (bool, error)
	CountInstallations(ctx context.Context, pluginID types.PluginID) (int, error)
	IsPluginPaused(ctx context.Context, pluginID types.PluginID) (bool, error)
	PausePlugin(ctx context.Context, pluginID types.PluginID, record itypes.PauseHistoryRecord) error
}

type ReportService struct {
	db     ReportServiceStorage
	logger *logrus.Logger
}

func NewReportService(db ReportServiceStorage, logger *logrus.Logger) (*ReportService, error) {
	if db == nil {
		return nil, fmt.Errorf("database storage cannot be nil")
	}
	if logger == nil {
		return nil, fmt.Errorf("logger cannot be nil")
	}
	return &ReportService{
		db:     db,
		logger: logger,
	}, nil
}

func (s *ReportService) SubmitReport(ctx context.Context, pluginID types.PluginID, publicKey, reason string) (*itypes.ReportSubmitResult, error) {
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
			return nil, fmt.Errorf("cooldown active: please wait %s before reporting again", remaining.Round(time.Minute))
		}
	}

	err = s.db.UpsertReport(ctx, pluginID, publicKey, reason, cooldown)
	if err != nil {
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

func (s *ReportService) evaluateAndPause(ctx context.Context, pluginID types.PluginID) error {
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

	return nil
}
