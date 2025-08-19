package types

import (
	"encoding/hex"
	"fmt"

	"github.com/google/uuid"
)

func isValidHexString(s string) bool {
	buf, err := hex.DecodeString(s)
	return err == nil && len(buf) == 32
}

// ReshareRequest is a struct that represents a request to reshare a vault
type ReshareRequest struct {
	Name             string   `json:"name"`               // name of the vault
	PublicKey        string   `json:"public_key"`         // public key ecdsa
	SessionID        string   `json:"session_id"`         // session id
	HexEncryptionKey string   `json:"hex_encryption_key"` // hex encryption key
	HexChainCode     string   `json:"hex_chain_code"`     // hex chain code
	LocalPartyId     string   `json:"local_party_id"`     // local party id
	OldParties       []string `json:"old_parties"`        // old parties
	Email            string   `json:"email"`
	PluginID         string   `json:"plugin_id"` // plugin id
}

func (req *ReshareRequest) IsValid() error {
	if req.Name == "" {
		return fmt.Errorf("name is required")
	}
	if req.SessionID == "" {
		return fmt.Errorf("session_id is required")
	}
	if _, err := uuid.Parse(req.SessionID); err != nil {
		return fmt.Errorf("session_id is not valid")
	}
	if req.HexEncryptionKey == "" {
		return fmt.Errorf("hex_encryption_key is required")
	}
	if !isValidHexString(req.HexEncryptionKey) {
		return fmt.Errorf("hex_encryption_key is not valid")
	}
	if req.HexChainCode == "" {
		return fmt.Errorf("hex_chain_code is required")
	}
	if !isValidHexString(req.HexChainCode) {
		return fmt.Errorf("hex_chain_code is not valid")
	}

	if len(req.OldParties) == 0 {
		return fmt.Errorf("old_parties is required")
	}

	return nil
}
