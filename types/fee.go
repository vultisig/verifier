package types

type TxType string

const (
	TxTypeDebit  TxType = "debit"
	TxTypeCredit TxType = "credit"
)

const (
	FeeTypeInstallationFee = "installation_fee"
	FeeSubscribtionFee     = "subscription_fee"
	FeeTxExecFee           = "transaction_execution_fee"
)
