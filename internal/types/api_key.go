package types

import (
	"time"
)

type APIKey struct {
	ID        string     `json:"id"`
	ApiKey    string     `json:"apiKey"`
	PluginID  string     `json:"pluginId"`
	Status    int64      `json:"status"`
	ExpiresAt *time.Time `json:"expires_at"`
}
