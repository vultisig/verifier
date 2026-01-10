package types

import (
	"time"

	"github.com/google/uuid"

	"github.com/vultisig/verifier/types"
)

type PluginReport struct {
	PluginID        types.PluginID `json:"plugin_id"`
	ReporterPubKey  string         `json:"reporter_public_key"`
	Reason          string         `json:"reason"`
	CreatedAt       time.Time      `json:"created_at"`
	LastReportedAt  time.Time      `json:"last_reported_at"`
	ReportCount     int            `json:"report_count"`
}

type PauseHistoryRecord struct {
	ID                uuid.UUID      `json:"id"`
	PluginID          types.PluginID `json:"plugin_id"`
	Action            string         `json:"action"`
	ReportCountWindow *int           `json:"report_count_window,omitempty"`
	ActiveUsers       *int           `json:"active_users,omitempty"`
	ThresholdRate     *float64       `json:"threshold_rate,omitempty"`
	Reason            *string        `json:"reason,omitempty"`
	TriggeredBy       *string        `json:"triggered_by,omitempty"`
	CreatedAt         time.Time      `json:"created_at"`
}

type ReportCreateRequest struct {
	Reason string `json:"reason" validate:"required"`
}

type ReportSubmitResult struct {
	Status       string `json:"status"`
	PluginPaused bool   `json:"plugin_paused"`
}
