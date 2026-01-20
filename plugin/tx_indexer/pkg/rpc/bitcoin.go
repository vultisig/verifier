package rpc

// Bitcoin is an alias for Utxo client configured for Bitcoin chain.
// It uses Blockchair REST API for transaction status lookups.
type Bitcoin = Utxo

// NewBitcoin creates a Bitcoin RPC client using Blockchair REST API.
// The baseURL should be a Blockchair-compatible endpoint (e.g., "https://api.vultisig.com/blockchair/bitcoin"
// or "https://api.blockchair.com/bitcoin").
//
// Note: This uses Blockchair REST API, not Bitcoin Core JSON-RPC.
// The URL format should be the base URL without trailing path components.
// For example: "https://api.vultisig.com/blockchair" (chainPath "bitcoin" is appended internally)
func NewBitcoin(baseURL string) (*Bitcoin, error) {
	return NewUtxo(baseURL, "bitcoin")
}
