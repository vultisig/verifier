package types

import (
	"github.com/google/uuid"
)

type BillingPolicy struct {
	ID        uuid.UUID `json:"id" validate:"required"`
	Type      string    `json:"type" validate:"required"` // "tx", "recurring" or "once"
	Frequency string    `json:"frequency"`                // only "monthly" for now
	StartDate string    `json:"start_date"`               // Number of a month, e.g., "1" for the first month. Only allow 1 for now
}

type PluginPolicy struct {
	ID            uuid.UUID       `json:"id" validate:"required"`
	PublicKey     string          `json:"public_key" validate:"required"`
	PluginID      PluginID        `json:"plugin_id" validate:"required"`
	PluginVersion string          `json:"plugin_version" validate:"required"`
	PolicyVersion string          `json:"policy_version" validate:"required"`
	Signature     string          `json:"signature" validate:"required"`
	Billing       []BillingPolicy `json:"billing" validate:"required"`
	Recipe        string          `json:"recipe" validate:"required"` // base64 encoded recipe protobuf bytes
	Active        bool            `json:"active" validate:"required"`
}
