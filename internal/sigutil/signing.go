package sigutil

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

// VerifyEthAddressSignature verifies if a message was signed by the owner of an Ethereum address
func VerifyEthAddressSignature(address common.Address, messageBytes []byte, signatureBytes []byte) (bool, error) {
	// Ensure signature is 65 bytes long (r, s, v)
	if len(signatureBytes) != 65 {
		return false, fmt.Errorf("invalid signature length: expected 65 bytes, got %d", len(signatureBytes))
	}

	sigCopy := make([]byte, 65)
	copy(sigCopy, signatureBytes)

	if sigCopy[64] >= 27 {
		sigCopy[64] -= 27
	}

	// Create the Ethereum prefixed message hash
	prefixedMessage := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(messageBytes), messageBytes)
	prefixedHash := crypto.Keccak256Hash([]byte(prefixedMessage))

	// Recover public key from signature
	pubkeyBytes, err := crypto.Ecrecover(prefixedHash.Bytes(), sigCopy)
	if err != nil {
		return false, fmt.Errorf("failed to recover public key: %w", err)
	}

	// Convert recovered pubkey to address
	recoveredPubKey, err := crypto.UnmarshalPubkey(pubkeyBytes)
	if err != nil {
		return false, fmt.Errorf("failed to unmarshal recovered public key: %w", err)
	}
	recoveredAddr := crypto.PubkeyToAddress(*recoveredPubKey)

	// Compare addresses
	return address == recoveredAddr, nil
}

// VerifyEIP712Signature verifies an EIP-712 typed data signature
func VerifyEIP712Signature(address common.Address, typedData apitypes.TypedData, signatureBytes []byte) (bool, error) {
	// Ensure signature is 65 bytes long (r, s, v)
	if len(signatureBytes) != 65 {
		return false, fmt.Errorf("invalid signature length: expected 65 bytes, got %d", len(signatureBytes))
	}

	sigCopy := make([]byte, 65)
	copy(sigCopy, signatureBytes)

	if sigCopy[64] >= 27 {
		sigCopy[64] -= 27
	}

	// Hash the typed data according to EIP-712
	domainSeparator, err := typedData.HashStruct("EIP712Domain", typedData.Domain.Map())
	if err != nil {
		return false, fmt.Errorf("failed to hash domain separator: %w", err)
	}

	messageHash, err := typedData.HashStruct(typedData.PrimaryType, typedData.Message)
	if err != nil {
		return false, fmt.Errorf("failed to hash message: %w", err)
	}

	// EIP-712: keccak256("\x19\x01" + domainSeparator + messageHash)
	rawData := append([]byte("\x19\x01"), domainSeparator...)
	rawData = append(rawData, messageHash...)
	digest := crypto.Keccak256Hash(rawData)

	// Recover public key from signature
	pubkeyBytes, err := crypto.Ecrecover(digest.Bytes(), sigCopy)
	if err != nil {
		return false, fmt.Errorf("failed to recover public key: %w", err)
	}

	// Convert recovered pubkey to address
	recoveredPubKey, err := crypto.UnmarshalPubkey(pubkeyBytes)
	if err != nil {
		return false, fmt.Errorf("failed to unmarshal recovered public key: %w", err)
	}
	recoveredAddr := crypto.PubkeyToAddress(*recoveredPubKey)

	return address == recoveredAddr, nil
}

// PluginUpdateTypedData creates the EIP-712 typed data for plugin updates
func PluginUpdateTypedData(pluginID, signer string, nonce, timestamp int64, updates []FieldUpdate) apitypes.TypedData {
	// Convert updates to []interface{} for the message
	updatesList := make([]interface{}, len(updates))
	for i, u := range updates {
		updatesList[i] = map[string]interface{}{
			"field":    u.Field,
			"oldValue": u.OldValue,
			"newValue": u.NewValue,
		}
	}

	return apitypes.TypedData{
		Types: apitypes.Types{
			"EIP712Domain": []apitypes.Type{
				{Name: "name", Type: "string"},
				{Name: "version", Type: "string"},
				{Name: "chainId", Type: "uint256"},
			},
			"PluginUpdate": []apitypes.Type{
				{Name: "pluginId", Type: "string"},
				{Name: "signer", Type: "address"},
				{Name: "nonce", Type: "uint256"},
				{Name: "timestamp", Type: "uint256"},
				{Name: "updates", Type: "FieldUpdate[]"},
			},
			"FieldUpdate": []apitypes.Type{
				{Name: "field", Type: "string"},
				{Name: "oldValue", Type: "string"},
				{Name: "newValue", Type: "string"},
			},
		},
		PrimaryType: "PluginUpdate",
		Domain: apitypes.TypedDataDomain{
			Name:    "Vultisig Developer Portal",
			Version: "1",
			ChainId: math.NewHexOrDecimal256(1),
		},
		Message: apitypes.TypedDataMessage{
			"pluginId":  pluginID,
			"signer":    signer,
			"nonce":     math.NewHexOrDecimal256(nonce),
			"timestamp": math.NewHexOrDecimal256(timestamp),
			"updates":   updatesList,
		},
	}
}

// FieldUpdate represents a single field change in a plugin update
type FieldUpdate struct {
	Field    string `json:"field"`
	OldValue string `json:"oldValue"`
	NewValue string `json:"newValue"`
}
