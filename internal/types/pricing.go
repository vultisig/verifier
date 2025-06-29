package types

import (
	"time"

	"github.com/google/uuid"
)

type Pricing struct {
	ID        uuid.UUID `json:"id" validate:"required"`
	Type      string    `json:"type" validate:"required,oneof=FREE SINGLE RECURRING PER_TX"`
	Frequency *string   `json:"frequency,omitempty" validate:"omitempty,oneof=ANNUAL MONTHLY WEEKLY"`
	Amount    float64   `json:"amount" validate:"gte=0"`
	Metric    string    `json:"metric" validate:"required,oneof=FIXED PERCENTAGE"`
	CreatedAt time.Time `json:"created_at" validate:"required"`
	UpdatedAt time.Time `json:"updated_at" validate:"required"`
}

type PricingCreateDto struct {
	Type      string  `json:"type" validate:"required,oneof=FREE SINGLE RECURRING PER_TX"`
	Frequency string  `json:"frequency,omitempty" validate:"omitempty,oneof=ANNUAL MONTHLY WEEKLY"`
	Amount    float64 `json:"amount" validate:"gte=0"`
	Metric    string  `json:"metric" validate:"required,oneof=FIXED PERCENTAGE"`
}
