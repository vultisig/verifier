package types

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"

	etypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/google/uuid"
	"github.com/vultisig/recipes/chain/evm/ethereum"
	vgcommon "github.com/vultisig/vultisig-go/common"
)

type HashFunction string

const (
	HashFunction_SHA256 HashFunction = "SHA256"
)

type KeysignRequest struct {
	PublicKey        string           `json:"public_key"` // public key, used to identify the backup file
	Messages         []KeysignMessage `json:"messages"`
	SessionID        string           `json:"session"`            // Session ID , it should be an UUID
	HexEncryptionKey string           `json:"hex_encryption_key"` // Hex encryption key, used to encrypt the keysign messages
	Parties          []string         `json:"parties"`            // parties to join the session
	PluginID         string           `json:"plugin_id"`          // plugin id
	PolicyID         uuid.UUID        `json:"policy_id"`          // policy id
}

type KeysignMessage struct {
	TxIndexerID  string         `json:"tx_indexer_id"` // Tx indexer uuid
	RawMessage   string         `json:"raw_message"`   // Raw message, used to decode the transaction
	Message      string         `json:"message"`
	Hash         string         `json:"hash"`
	HashFunction HashFunction   `json:"hash_function"`
	Chain        vgcommon.Chain `json:"chain"`
}

// IsValid checks if the keysign request is valid
func (r KeysignRequest) IsValid() error {
	if r.PublicKey == "" {
		return errors.New("invalid public key ECDSA")
	}
	if len(r.Messages) == 0 {
		return errors.New("invalid messages")
	}
	for _, m := range r.Messages {
		_, err := base64.StdEncoding.DecodeString(m.Message)
		if err != nil {
			return errors.New("message is not base64 encoded")
		}
	}
	if r.SessionID == "" {
		return errors.New("invalid session")
	}
	if r.HexEncryptionKey == "" {
		return errors.New("invalid hex encryption key")
	}

	return nil
}

type PluginKeysignRequest struct {
	KeysignRequest
	Transaction     string `json:"transactions"`
	TransactionType string `json:"transaction_type"`
	// SignBytes is required for Cosmos chains where signBytes cannot be derived from Transaction.
	// For non-Cosmos chains, this field is ignored.
	SignBytes string `json:"sign_bytes,omitempty"`
}

func NewPluginKeysignRequestEvm(policy PluginPolicy, txToTrack string, chain vgcommon.Chain, tx []byte) (
	*PluginKeysignRequest, error) {
	ethEvmID, err := chain.EvmID()
	if err != nil {
		return nil, fmt.Errorf("failed to get EVM ID for chain %s: %w", chain, err)
	}

	txData, e := ethereum.DecodeUnsignedPayload(tx)
	if e != nil {
		return nil, fmt.Errorf("ethereum.DecodeUnsignedPayload: %w", e)
	}
	txHashToSign := etypes.LatestSignerForChainID(ethEvmID).Hash(etypes.NewTx(txData))
	msgHash := sha256.Sum256(txHashToSign.Bytes())

	return &PluginKeysignRequest{
		KeysignRequest: KeysignRequest{
			PublicKey: policy.PublicKey,
			Messages: []KeysignMessage{
				{
					TxIndexerID:  txToTrack,
					Message:      base64.StdEncoding.EncodeToString(txHashToSign.Bytes()),
					Chain:        chain,
					Hash:         base64.StdEncoding.EncodeToString(msgHash[:]),
					HashFunction: HashFunction_SHA256,
				},
			},
			PolicyID: policy.ID,
			PluginID: policy.PluginID,
		},
		Transaction: base64.StdEncoding.EncodeToString(tx),
	}, nil
}
