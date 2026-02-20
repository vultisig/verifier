package types

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/vultisig/verifier/types"
)

const (
	ProgressPercent = "percent"
	ProgressCounter = "counter"
)

type Plugin struct {
	ID             types.PluginID  `json:"id" validate:"required"`
	Title          string          `json:"title" validate:"required"`
	Description    string          `json:"description" validate:"required"`
	ServerEndpoint string          `json:"server_endpoint" validate:"required"`
	Category       PluginCategory  `json:"category_id" validate:"required"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
	Pricing        []types.Pricing `json:"pricing,omitempty"`
	LogoURL        string          `json:"logo_url,omitempty"`
	ThumbnailURL   string          `json:"thumbnail_url,omitempty"`
	BannerURL      string          `json:"banner_url,omitempty"`
	Images         []PluginImage   `json:"images,omitempty"`
	FAQs           []FAQItem       `json:"faqs,omitempty"`
	Features       []string        `json:"features,omitempty"`
	Audited        bool            `json:"audited"`
	RatesCount     int             `json:"rates_count"`
	AvgRating      float64         `json:"avg_rating"`
	Installations  int             `json:"installations"`
}

type FAQItem struct {
	Question string `json:"question"`
	Answer   string `json:"answer"`
}

type PluginImage struct {
	ID        string `json:"id"`
	URL       string `json:"url"`
	S3Key     string `json:"-"`
	SortOrder int    `json:"sort_order"`
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
	CategoryID     uuid.UUID                    `json:"category_id" validate:"required"`
	PricingData    []types.PricingCreateDataDto `json:"pricing_data" validate:"required"`
}

// using references on struct fields allows us to process partially field DTOs
type PluginUpdateDto struct {
	Title          string                       `json:"title"`
	Description    string                       `json:"description"`
	Metadata       json.RawMessage              `json:"metadata"`
	ServerEndpoint string                       `json:"server_endpoint"`
	PricingData    []types.PricingCreateDataDto `json:"pricing_data"`
	CategoryID     uuid.UUID                    `json:"category_id"`
}

type PluginPolicyPaginatedList struct {
	Policies   []types.PluginPolicy `json:"policies" validate:"required"`
	TotalCount int                  `json:"total_count" validate:"required"`
}

type Progress struct {
	Kind  string `json:"kind"`
	Value uint32 `json:"value"`
}

type PluginPolicyResponse struct {
	types.PluginPolicy
	Progress Progress `json:"progress"`
}

type PluginPolicyResponsePaginatedList struct {
	Policies   []PluginPolicyResponse `json:"policies"`
	TotalCount int                    `json:"total_count"`
}

type PluginTotalCount struct {
	ID         types.PluginID `json:"id" validate:"required"`
	TotalCount int            `json:"total_count" validate:"required"`
}

type RecipeFunctions struct {
	ID        string   `json:"id"`
	Functions []string `json:"functions"`
}
