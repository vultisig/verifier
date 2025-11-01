package types

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
