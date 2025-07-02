package types

import "github.com/google/uuid"

type FeeDto struct {
	ID          uuid.UUID `json:"id" validate:"required"`
	Amount      int       `json:"amount" validate:"required"`
	ChargedAt   string    `json:"charged_on" validate:"required"` // "tx" or "recurring"
	Collected   bool      `json:"collected" validate:"required"`  // true if the fee is collected, false if it's just a record
	CollectedAt string    `json:"collected_at"`                   // timestamp when the fee was collected
	PublicKey   string    `json:"public_key" validate:"required"`
	PolicyId    uuid.UUID `json:"policy_id" validate:"required"`
	PluginId    string    `json:"plugin_id" validate:"required"`
}

type FeeHistoryDto struct {
	// PolicyId              uuid.UUID `json:"policy_id" validate:"required"`
	Fees                  []FeeDto `json:"fees" validate:"required"`
	TotalFeesIncurred     int      `json:"total_fees_incurred" validate:"required"`     // Total fees incurred in the smallest unit, e.g., "1000000" for 0.01 VULTI
	FeesPendingCollection int      `json:"fees_pending_collection" validate:"required"` // Total fees pending collection in the smallest unit, e.g., "1000000" for 0.01 VULTI
}
