package types

import (
	"time"

	vtypes "github.com/vultisig/verifier/types"
)

type PluginOwner struct {
	PluginID         vtypes.PluginID `json:"plugin_id"`
	PublicKey        string         `json:"public_key"`
	Active           bool           `json:"active"`
	AddedVia         string         `json:"added_via"`
	AddedByPublicKey string         `json:"added_by_public_key,omitempty"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
}
