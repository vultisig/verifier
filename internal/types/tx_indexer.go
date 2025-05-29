package types

type TxStatus string
type TxOnChainStatus string

const (
	TxProposed       TxStatus        = "PROPOSED"
	TxVerified       TxStatus        = "VERIFIED"
	TxSigned         TxStatus        = "SIGNED"
	TxOnChainPending TxOnChainStatus = "PENDING"
	TxOnChainSuccess TxOnChainStatus = "SUCCESS"
	TxOnChainFail    TxOnChainStatus = "FAIL"
)

type Chain string

const (
	Bitcoin  Chain = "bitcoin"
	Ethereum Chain = "ethereum"
)
