package types

import (
	"time"

	"github.com/vultisig/verifier/types"
)

type APIKey struct {
	ID        string         `json:"id"`
	ApiKey    string         `json:"apiKey"`
	PluginID  types.PluginID `json:"pluginId"`
	Status    int64          `json:"status"`
	ExpiresAt *time.Time     `json:"expires_at"`
}
