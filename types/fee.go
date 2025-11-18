package types

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type TxType string

const (
	TxTypeDebit  TxType = "debit"
	TxTypeCredit TxType = "credit"
)

const (
	FeeTypeInstallationFee = "installation_fee"
	FeeSubscriptionFee     = "subscription_fee"
	FeeTxExecFee           = "transaction_execution_fee"
)

type CreditMetadata struct {
	DebitFeeID uint64 `json:"debit_fee_id"` // ID of the debit transaction
	TxHash     string `json:"tx_hash"`      // Transaction hash in blockchain
	Network    string `json:"network"`      // Blockchain network (e.g., "ethereum", "polygon")
}

// UserFeeStatus represents the fee status and balance for a user
type UserFeeStatus struct {
	PublicKey    string `json:"public_key"`
	Balance      int64  `json:"balance"` // Current balance (can be negative)
	UnpaidAmount int64  `json:"unpaid_amount"`
	Fees         []*Fee `json:"fees"`
}

type Fee struct {
	ID             uint64          `json:"id"`         // The unique id of the fee incurred
	PolicyID       uuid.UUID       `json:"policy_id"`  // The policy ID that has incurred the fee
	PublicKey      string          `json:"public_key"` // The public key "account" connected to the fee
	TxType         TxType          `json:"transaction_type"`
	Amount         uint64          `json:"amount"` // The amount of the fee in the smallest unit, e.g., "1000000" for 0.01 VULTI
	CreatedAt      time.Time       `json:"created_at"`
	FeeType        string          `json:"fee_type"`
	Metadata       json.RawMessage `json:"metadata"`
	UnderlyingType string          `json:"underlying_type"`
	UnderlyingID   string          `json:"underlying_id"`
}
