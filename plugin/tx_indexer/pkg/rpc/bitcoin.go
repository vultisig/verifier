package rpc

// NewBitcoin creates a Bitcoin RPC client using Blockchair API.
// This uses the same Blockchair REST API as other UTXO chains.
func NewBitcoin(baseURL string) (*Utxo, error) {
	return NewUtxo(baseURL, "bitcoin")
}
