package types

import (
	"time"

	"github.com/google/uuid"
	"github.com/vultisig/verifier/types"
)

type PricingFrequency string

const (
	PricingFrequencyDaily    PricingFrequency = "daily"
	PricingFrequencyWeekly   PricingFrequency = "weekly"
	PricingFrequencyBiweekly PricingFrequency = "biweekly"
	PricingFrequencyMonthly  PricingFrequency = "monthly"
)

type PricingType string

const (
	PricingTypeOnce      PricingType = "once"
	PricingTypeRecurring PricingType = "recurring"
	PricingTypePerTx     PricingType = "per-tx"
)

type PricingMetric string

const (
	PricingMetricFixed PricingMetric = "fixed"
)

type PricingAsset string

const (
	PricingAssetUSDC PricingAsset = "usdc"
)

type Pricing struct {
	ID        uuid.UUID         `json:"id" validate:"required"`
	Type      PricingType       `json:"type" validate:"required"`
	Frequency *PricingFrequency `json:"frequency,omitempty"`
	Amount    uint64            `json:"amount" validate:"gte=0"`
	Asset     PricingAsset      `json:"asset" validate:"required"`
	Metric    PricingMetric     `json:"metric" validate:"required"`
	CreatedAt time.Time         `json:"created_at" validate:"required"`
	UpdatedAt time.Time         `json:"updated_at" validate:"required"`
	PluginID  types.PluginID    `json:"plugin_id" validate:"required"`
}
type PricingCreateDataDto struct {
	Type      PricingType       `json:"type" validate:"required"`
	Frequency *PricingFrequency `json:"frequency,omitempty" validate:"omitempty"`
	Amount    uint64            `json:"amount" validate:"gte=0"`
	Metric    PricingMetric     `json:"metric" validate:"required"`
}
type PricingCreateDto struct {
	PricingCreateDataDto
	PluginID types.PluginID `json:"plugin_id" validate:"required"`
}
