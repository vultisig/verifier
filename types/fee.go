package types

type TxType string

const (
	TxTypeDebit  TxType = "debit"
	TxTypeCredit TxType = "credit"
)

const FeeTypeInstallationFee = "installation_fee"
