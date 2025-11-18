package types

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/vultisig/verifier/types"
)

type Plugin struct {
	ID             types.PluginID  `json:"id" validate:"required"`
	Title          string          `json:"title" validate:"required"`
	Description    string          `json:"description" validate:"required"`
	ServerEndpoint string          `json:"server_endpoint" validate:"required"`
	Category       PluginCategory  `json:"category_id" validate:"required"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
	LogoURL        string          `json:"logo_url,omitempty"` // New field, should be validated once migration is done.
	Pricing        []types.Pricing `json:"pricing,omitempty"`  // New field for multiple pricing options
}

// PluginWithRatings is used for API responses that include rating statistics
type PluginWithRatings struct {
	Plugin
	Ratings []PluginRatingDto `json:"ratings,omitempty"`
}

type PluginFilters struct {
	Term       *string    `json:"term"`
	TagID      *uuid.UUID `json:"tag_id"`
	CategoryID *string    `json:"category_id"`
}

type PluginsPaginatedList struct {
	Plugins    []Plugin `json:"plugins"`
	TotalCount int      `json:"total_count"`
}

type PluginCreateDto struct {
	Type           string                       `json:"type" validate:"required"`
	Title          string                       `json:"title" validate:"required"`
	Description    string                       `json:"description" validate:"required"`
	Metadata       json.RawMessage              `json:"metadata" validate:"required"`
	ServerEndpoint string                       `json:"server_endpoint" validate:"required"`
	LogoURL        string                       `json:"logo_url"`
	CategoryID     uuid.UUID                    `json:"category_id" validate:"required"`
	PricingData    []types.PricingCreateDataDto `json:"pricing_data" validate:"required"`
}

// using references on struct fields allows us to process partially field DTOs
type PluginUpdateDto struct {
	Title          string                       `json:"title"`
	Description    string                       `json:"description"`
	Metadata       json.RawMessage              `json:"metadata"`
	ServerEndpoint string                       `json:"server_endpoint"`
	LogoURL        string                       `json:"logo_url"`
	PricingData    []types.PricingCreateDataDto `json:"pricing_data"`
	CategoryID     uuid.UUID                    `json:"category_id"`
}

type PluginPolicyPaginatedList struct {
	Policies   []types.PluginPolicy `json:"policies" validate:"required"`
	TotalCount int                  `json:"total_count" validate:"required"`
}

type PluginTotalCount struct {
	ID         types.PluginID `json:"id" validate:"required"`
	TotalCount int            `json:"total_count" validate:"required"`
}

type RecipeFunctions struct {
	ID        string   `json:"id"`
	Functions []string `json:"functions"`
}
