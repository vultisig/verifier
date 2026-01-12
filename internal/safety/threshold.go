package safety

import (
	"math"
	"time"
)

const (
	// Rolling window for counting reports (reportsInWindow)
	ReportsWindowDuration = 7 * 24 * time.Hour

	MinReportsToTriggerCheck = 5

	// Threshold formula: max(MinReportRate, SqrtActiveUsersScaleFactor/sqrt(activeUsers))
	MinReportRate              = 0.02
	SqrtActiveUsersScaleFactor = 0.5

	// Cooldown before user can update last_reported_at; stricter when paused
	ReportCooldown       = 24 * time.Hour
	ReportPausedCooldown = 7 * 24 * time.Hour
)

func CalculateThreshold(activeUsers int) float64 {
	if activeUsers <= 0 {
		return 1.0
	}
	return math.Max(MinReportRate, SqrtActiveUsersScaleFactor/math.Sqrt(float64(activeUsers)))
}

func ShouldAutoPause(reportsInWindow, activeUsers int) (should bool, rate, threshold float64) {
	if reportsInWindow < MinReportsToTriggerCheck {
		return false, 0, 0
	}
	threshold = CalculateThreshold(activeUsers)
	rate = float64(reportsInWindow) / float64(max(activeUsers, 1))
	return rate >= threshold, rate, threshold
}
