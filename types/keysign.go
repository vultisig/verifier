package types

import (
	"errors"

	"github.com/google/uuid"
	"github.com/vultisig/verifier/common"
)

type HashFunction string

const (
	HashFunction_SHA256 HashFunction = "SHA256"
)

type KeysignRequest struct {
	PublicKey string           `json:"public_key"` // public key, used to identify the backup file
	Messages  []KeysignMessage `json:"messages"`
	SessionID string           `json:"session"`   // Session ID , it should be an UUID
	Parties   []string         `json:"parties"`   // parties to join the session
	PluginID  string           `json:"plugin_id"` // plugin id
	PolicyID  uuid.UUID        `json:"policy_id"` // policy id
}

type KeysignMessage struct {
	Message      string       `json:"message"`
	Hash         string       `json:"hash"`
	HashFunction HashFunction `json:"hash_function"`
	Chain        common.Chain `json:"chain"`
}

// IsValid checks if the keysign request is valid
func (r KeysignRequest) IsValid() error {
	if r.PublicKey == "" {
		return errors.New("invalid public key ECDSA")
	}
	if len(r.Messages) == 0 {
		return errors.New("invalid messages")
	}
	if r.SessionID == "" {
		return errors.New("invalid session")
	}

	return nil
}

type PluginKeysignRequest struct {
	KeysignRequest
	Transaction     string `json:"transactions"`
	PolicyID        string `json:"policy_id"`
	TransactionType string `json:"transaction_type"`
}
