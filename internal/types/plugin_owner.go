package types

import (
	"time"
)

type PluginOwnerRole string

const (
	PluginOwnerRoleAdmin PluginOwnerRole = "admin"
)

type PluginOwnerAddedVia string

const (
	PluginOwnerAddedViaBootstrap    PluginOwnerAddedVia = "bootstrap_plugin_key"
	PluginOwnerAddedViaOwnerAPI     PluginOwnerAddedVia = "owner_api"
	PluginOwnerAddedViaAdminCLI     PluginOwnerAddedVia = "admin_cli"
	PluginOwnerAddedViaMagicLink    PluginOwnerAddedVia = "magic_link"
	PluginOwnerAddedViaPortalCreate PluginOwnerAddedVia = "portal_create"
)

type PluginOwner struct {
	PluginID         string              `json:"plugin_id"`
	PublicKey        string              `json:"public_key"`
	Active           bool                `json:"active"`
	Role             PluginOwnerRole     `json:"role"`
	AddedVia         PluginOwnerAddedVia `json:"added_via"`
	AddedByPublicKey string              `json:"added_by_public_key,omitempty"`
	CreatedAt        time.Time           `json:"created_at"`
	UpdatedAt        time.Time           `json:"updated_at"`
}
