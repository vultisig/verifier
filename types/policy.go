package types

import (
	"github.com/google/uuid"
)

type PluginPolicy struct {
	ID            uuid.UUID `json:"id" validate:"required"`
	PublicKey     string    `json:"public_key" validate:"required"`
	PluginID      PluginID  `json:"plugin_id" validate:"required"`
	PluginVersion string    `json:"plugin_version" validate:"required"`
	PolicyVersion string    `json:"policy_version" validate:"required"`
	Signature     string    `json:"signature" validate:"required"`
	Recipe        string    `json:"recipe" validate:"required"` // base64 encoded recipe protobuf bytes
	Active        bool      `json:"active" validate:"required"`
}
