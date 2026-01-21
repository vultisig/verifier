package types

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/vultisig/verifier/types"
)

type Plugin struct {
	ID             types.PluginID  `json:"id" yaml:"id" validate:"required"`
	Title          string          `json:"title" yaml:"title" validate:"required"`
	Description    string          `json:"description" yaml:"description" validate:"required"`
	ServerEndpoint string          `json:"server_endpoint" yaml:"server_endpoint" validate:"required"`
	Category       PluginCategory  `json:"category_id" yaml:"category" validate:"required"`
	CreatedAt      time.Time       `json:"created_at" yaml:"-"`
	UpdatedAt      time.Time       `json:"updated_at" yaml:"-"`
	Pricing        []types.Pricing `json:"pricing,omitempty" yaml:"-"`                             // New field for multiple pricing options
	LogoURL        string          `json:"logo_url,omitempty" yaml:"logo_url,omitempty"`           // New field, should be validated once plugins have this data in db.
	ThumbnailURL   string          `json:"thumbnail_url,omitempty" yaml:"thumbnail_url,omitempty"` // New field, should be validated once plugins have this data in db.
	Images         []PluginImage   `json:"images,omitempty" yaml:"images,omitempty"`               // New field, should be validated once plugins have this data in db.
	FAQs           []FAQItem       `json:"faqs,omitempty" yaml:"faqs,omitempty"`
	Features       []string        `json:"features,omitempty" yaml:"features,omitempty"`
	Audited        bool            `json:"audited" yaml:"audited,omitempty"`
	RatesCount     int             `json:"rates_count" yaml:"-"`
	AvgRating      float64         `json:"avg_rating" yaml:"-"`
	Installations  int             `json:"installations" yaml:"-"`
}

type FAQItem struct {
	Question string `json:"question" yaml:"question,omitempty"`
	Answer   string `json:"answer"  yaml:"answer,omitempty"`
}

type PluginImage struct {
	ID        string `json:"id" yaml:"id,omitempty"`
	URL       string `json:"url" yaml:"url"`
	S3Key     string `json:"-" yaml:"s3_key,omitempty"`
	Caption   string `json:"caption" yaml:"caption,omitempty"`
	AltText   string `json:"alt_text" yaml:"alt_text,omitempty"`
	SortOrder int    `json:"sort_order" yaml:"sort_order,omitempty"`
	ZIndex    int    `json:"z_index" yaml:"z_index,omitempty"`
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
