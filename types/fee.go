package types

import (
<<<<<<< HEAD
	"fmt"
=======
>>>>>>> 19a6210 (✨ Treasury Accounting Logic)
	"time"

	"github.com/google/uuid"
	rtypes "github.com/vultisig/recipes/types"
)

<<<<<<< HEAD
type FeeType string

const (
	FeeTypeDebit  FeeType = "debit"
	FeeTypeCredit FeeType = "credit"
)

type Fee struct {
	ID        uuid.UUID `json:"id"`         // The unique id of the fee incurred
	PublicKey string    `json:"public_key"` // The public key "account" connected to the fee
	Amount    uint64    `json:"amount"`     // The amount of the fee in the smallest unit, e.g., "1000000" for 0.01 VULTI
	CreatedAt time.Time `json:"created_at"` // The date and time the fee was created
	Ref       string    `json:"ref"`        // Reference to the external or internal reference, comma separated list of format: "type:id"
	Type      FeeType   `json:"type"`       // The type of fee (debit or credit)
}

// FeeDebitType represents the type of fee debit
type FeeDebitSubtypeType string

const (
	FeeDebitSubtypeTypeFee      FeeDebitSubtypeType = "fee"
	FeeDebitSubtypeTypeFailedTx FeeDebitSubtypeType = "failed_tx"
)

// FeeDebit represents a fee debit record - charges incurred by users
type FeeDebit struct {
	Fee                                       // Inherits base fee fields
	Subtype               FeeDebitSubtypeType `json:"subtype"`                  // Type of debit (fee, failed_tx)
	PluginPolicyBillingID *uuid.UUID          `json:"plugin_policy_billing_id"` // Reference to billing policy (nullable)
	ChargedAt             time.Time           `json:"charged_at"`               // Date when the fee was charged
}

// FeeCreditsType represents the type of fee credit
type FeeCreditSubtypeType string

const (
	FeeCreditSubtypeTypeFeeTransacted FeeCreditSubtypeType = "fee_transacted"
)

// FeeCredit represents a fee credit record - refunds or payments to users
type FeeCredit struct {
	Fee                          // Inherits base fee fields
	Subtype FeeCreditSubtypeType `json:"subtype"` // Type of credit (fee_transacted)
}

type FeeBatchStatus string

const (
	FeeBatchStatusDraft     FeeBatchStatus = "draft"
	FeeBatchStatusSent      FeeBatchStatus = "sent"
	FeeBatchStatusCompleted FeeBatchStatus = "completed"
	FeeBatchStatusFailed    FeeBatchStatus = "failed"
)

// FeeBatch represents a batch of fees collected in a single transaction
type FeeBatch struct {
	ID        uuid.UUID      `json:"id"`         // Unique identifier for the batch
	CreatedAt time.Time      `json:"created_at"` // When the batch was created
	TxHash    *string        `json:"tx_hash"`    // Transaction hash where fees were collected
	Status    FeeBatchStatus `json:"status"`     // Status of the batch
}

// FeeBatchMembers represents the many-to-many relationship between fee batches and individual fees
type FeeBatchMembers struct {
	FeeBatchID uuid.UUID `json:"fee_batch_id"` // Reference to the fee batch
	FeeID      uuid.UUID `json:"fee_id"`       // Reference to the individual fee
=======
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
>>>>>>> 19a6210 (✨ Treasury Accounting Logic)
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

<<<<<<< HEAD
// This is used to populate the Billing field of a PluginPolicy from the Recipe field. It does not validate this information against the plugin pricing.
func (p *PluginPolicy) ParseBillingFromRecipe() error {

	p.Billing = []BillingPolicy{}

	recipe, err := p.GetRecipe()
	if err != nil {
		return fmt.Errorf("failed to get recipe: %w", err)
	}

	for _, feePolicy := range recipe.FeePolicies {
		if feePolicy.Id == "" {
			feePolicy.Id = uuid.New().String()
		}
		id, err := uuid.Parse(feePolicy.Id)
		if err != nil {
			return fmt.Errorf("failed to parse fee policy ID: %w", err)
		}

		var feeType PricingType
		switch feePolicy.Type {
		case rtypes.FeeType_FEE_TYPE_UNSPECIFIED:
			return fmt.Errorf("fee type is unspecified")
		case rtypes.FeeType_ONCE:
			feeType = PricingTypeOnce
		case rtypes.FeeType_TRANSACTION:
			feeType = PricingTypePerTx
		case rtypes.FeeType_RECURRING:
			feeType = PricingTypeRecurring
		default:
			return fmt.Errorf("invalid fee type: %v", feePolicy.Type)
		}

		var pricingFrequency *PricingFrequency
		if feeType == PricingTypeRecurring {
			switch feePolicy.Frequency {
			case rtypes.BillingFrequency_BILLING_FREQUENCY_UNSPECIFIED:
				return fmt.Errorf("invalid frequency: %v", feePolicy.Frequency)
			case rtypes.BillingFrequency_DAILY:
				_freq := PricingFrequencyDaily
				pricingFrequency = &_freq
			case rtypes.BillingFrequency_WEEKLY:
				_freq := PricingFrequencyWeekly
				pricingFrequency = &_freq
			case rtypes.BillingFrequency_BIWEEKLY:
				_freq := PricingFrequencyBiweekly
				pricingFrequency = &_freq
			case rtypes.BillingFrequency_MONTHLY:
				_freq := PricingFrequencyMonthly
				pricingFrequency = &_freq
			default:
				return fmt.Errorf("invalid frequency: %v", feePolicy.Frequency)
			}
		}

		p.Billing = append(p.Billing, BillingPolicy{
			ID:        id,
			Type:      feeType,
			Frequency: pricingFrequency,
			StartDate: feePolicy.StartDate.AsTime(),
			Amount:    uint64(feePolicy.Amount),
			Asset:     "usdc", // Multiple currencies not currently supported in fee policy recipes or elsewhere so for now we can hard code it, and later, extract from the protobuf encoded fee policies
		})
	}
	return nil
=======
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
>>>>>>> 19a6210 (✨ Treasury Accounting Logic)
}
