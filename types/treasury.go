package types

import (
	"time"

	"github.com/google/uuid"
)

// Treasury ledger type (debit/credit)
type TreasuryLedgerType string

const (
	TreasuryLedgerTypeDebit  TreasuryLedgerType = "debit"
	TreasuryLedgerTypeCredit TreasuryLedgerType = "credit"
)

// Treasury ledger entry subtype (specific operation type)
type TreasuryLedgerEntryType string

const (
	TreasuryLedgerEntryTypeFeeCollection   TreasuryLedgerEntryType = "fee_collection"
	TreasuryLedgerEntryTypeDeveloperPayout TreasuryLedgerEntryType = "developer_payout"
	TreasuryLedgerEntryTypeFailedTx        TreasuryLedgerEntryType = "failed_tx"
)

// Treasury batch status
type TreasuryBatchStatus string

const (
	TreasuryBatchStatusDraft     TreasuryBatchStatus = "draft"
	TreasuryBatchStatusSent      TreasuryBatchStatus = "sent"
	TreasuryBatchStatusCompleted TreasuryBatchStatus = "completed"
	TreasuryBatchStatusFailed    TreasuryBatchStatus = "failed"
)

// Treasury batch represents a collection of fees
type TreasuryBatch struct {
	ID          uuid.UUID           `json:"id"`
	CreatedAt   time.Time           `json:"created_at"`
	TxHash      *string             `json:"tx_hash"`
	DeveloperID uuid.UUID           `json:"developer_id"`
	Status      TreasuryBatchStatus `json:"status"`
}

// Treasury batch member links treasury ledger records to batches
type TreasuryBatchMember struct {
	BatchID                uuid.UUID `json:"batch_id"`
	TreasuryLedgerRecordID uuid.UUID `json:"treasury_ledger_record_id"`
}

// Treasury batch members view aggregates batch information
type TreasuryBatchMembersView struct {
	BatchID     uuid.UUID `json:"batch_id"`
	RecordCount uint64    `json:"record_count"`
	TotalAmount uint64    `json:"total_amount"` // We cannot create a batch if the balance is 0 or less and so it must always be positive
}

// Treasury ledger record (base ledger entry)
type TreasuryLedgerRecord struct {
	ID          uuid.UUID               `json:"id"`
	Amount      int64                   `json:"amount"`    // Always positive in database
	Type        TreasuryLedgerType      `json:"type"`      // 'debit' or 'credit'
	Subtype     TreasuryLedgerEntryType `json:"subtype"`   // Specific operation type
	Reference   string                  `json:"reference"` // External references
	CreatedAt   time.Time               `json:"created_at"`
	DeveloperID uuid.UUID               `json:"developer_id"`
}

// Treasury ledger fee batch collection (debit entry)
type TreasuryLedgerFeeCollection struct {
	TreasuryLedgerRecord
	FeeBatchID uuid.UUID `json:"fee_batch_id"`
}

// Treasury ledger developer payout (credit entry)
type TreasuryLedgerDeveloperPayout struct {
	TreasuryLedgerRecord
	TreasuryBatchID uuid.UUID `json:"treasury_batch_id"`
}

// Treasury ledger failed transaction (debit entry)
type TreasuryLedgerFailedTx struct {
	TreasuryLedgerRecord
	TreasuryBatchID uuid.UUID `json:"treasury_batch_id"`
}

type Developer struct {
	ID        uuid.UUID `json:"id"`
	PublicKey string    `json:"public_key"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
