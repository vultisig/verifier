package types

import (
	"github.com/google/uuid"
	vtypes "github.com/vultisig/verifier/types"
)

type FeeDto struct {
	ID        uuid.UUID `json:"id" validate:"required"`
	Amount    uint64    `json:"amount" validate:"required"`
	ChargedAt string    `json:"charged_on" validate:"required"` // "tx" or "recurring"
	PublicKey string    `json:"public_key" validate:"required"`
	PolicyId  uuid.UUID `json:"policy_id" validate:"required"`
	PluginId  string    `json:"plugin_id" validate:"required"`
}

type FeeCreditDto struct {
	ID        uuid.UUID                   `json:"id" validate:"required"`
	Subtype   vtypes.FeeCreditSubtypeType `json:"subtype" validate:"required"`
	Amount    uint64                      `json:"amount" validate:"required"`
	ChargedAt string                      `json:"charged_on" validate:"required"`
	PublicKey string                      `json:"public_key" validate:"required"`
	PolicyId  uuid.UUID                   `json:"policy_id" validate:"required"`
	PluginId  string                      `json:"plugin_id" validate:"required"`
}

type FeeHistoryDto struct {
	PublicKey string `json:"public_key" validate:"required"`
	Fees      uint64 `json:"fees" validate:"required"`
}

type FeeBatchRequest struct {
	PublicKey string    `json:"public_key" validate:"required"`
	Amount    uint64    `json:"amount" validate:"required"`
	BatchID   uuid.UUID `json:"batch_id" validate:"required"`
}
