package types

import (
	"time"

	"github.com/google/uuid"
	rtypes "github.com/vultisig/recipes/types"
)

type Fee struct {
	ID                    uuid.UUID  `json:"id"`                       // The unique id of the fee incurred
	PublicKey             string     `json:"public_key"`               // The public key "account" connected to the fee
	PluginID              PluginID   `json:"plugin_id"`                // The plugin ID that has incurred the fee
	PolicyID              uuid.UUID  `json:"policy_id"`                // The policy ID that has incurred the fee
	PluginPolicyBillingID uuid.UUID  `json:"plugin_policy_billing_id"` // The plugin policy billing ID that has incurred the fee. This is because a plugin policy may have several billing "rules" associated with it.
	TransactionID         uuid.UUID  `json:"transaction_id"`           // The transaction ID that has incurred the fee
	Amount                uint64     `json:"amount"`                   // The amount of the fee in the smallest unit, e.g., "1000000" for 0.01 VULTI
	Type                  string     `json:"type"`                     // "tx", "recurring" or "once". Only availble on the fees_view table
	CreatedAt             time.Time  `json:"created_at"`
	ChargedAt             time.Time  `json:"charged_at"`
	CollectedAt           *time.Time `json:"collected_at"`
}

type BillingPolicyProto struct {
	ID        *uuid.UUID              `json:"id" validate:"required"`
	Type      rtypes.FeeType          `json:"type" validate:"required"`
	Frequency rtypes.BillingFrequency `json:"frequency"`
	StartDate time.Time               `json:"start_date"`                 // Number of a month, e.g., "1" for the first month. Only allow 1 for now
	Amount    uint64                  `json:"amount" validate:"required"` // Amount in the smallest unit, e.g., "1000000" for 0.01 VULTI
	Asset     string                  `json:"asset"`                      // The asset that the fee is denominated in, e.g., "usdc"
}

type BillingPolicy struct {
	ID        uuid.UUID         `json:"id" validate:"required"`
	Type      PricingType       `json:"type" validate:"required"`
	Frequency *PricingFrequency `json:"frequency"`
	StartDate time.Time         `json:"start_date"`                 // Number of a month, e.g., "1" for the first month. Only allow 1 for now
	Amount    uint64            `json:"amount" validate:"required"` // Amount in the smallest unit, e.g., "1000000" for 0.01 VULTI
	Asset     string            `json:"asset"`                      // The asset that the fee is denominated in, e.g., "usdc"
}

type TreasuryLedgerEntryType string

const (
	TreasuryLedgerEntryTypeFeeCredit        TreasuryLedgerEntryType = "fee_credit"
	TreasuryLedgerEntryTypeDeveloperPayout  TreasuryLedgerEntryType = "developer_payout"
	TreasuryLedgerEntryTypeRefund           TreasuryLedgerEntryType = "refund"
	TreasuryLedgerEntryTypeCreditAdjustment TreasuryLedgerEntryType = "credit_adjustment"
	TreasuryLedgerEntryTypeDebitAdjustment  TreasuryLedgerEntryType = "debit_adjustment"
)

type TreasuryLedgerRecord struct {
	ID     uuid.UUID               `json:"id" validate:"required"`
	Amount uint64                  `json:"amount" validate:"required"` // Amount in the smallest unit, e.g., "1000000" for 0.01 VULTI
	Type   TreasuryLedgerEntryType `json:"type" validate:"required"`

	FeeID       *uuid.UUID `json:"fee_id"`
	DeveloperID *uuid.UUID `json:"developer_id"`
	TxHash      string     `json:"tx_id"`
	Reference   string     `json:"reference"` //Used for any external references

	CreatedAt time.Time `json:"created_at"`
}
