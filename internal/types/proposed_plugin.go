package types

import (
	"time"

	"github.com/google/uuid"
)

type ProposedPluginStatus string

const (
	ProposedPluginStatusSubmitted ProposedPluginStatus = "submitted"
	ProposedPluginStatusApproved  ProposedPluginStatus = "approved"
	ProposedPluginStatusListed    ProposedPluginStatus = "listed"
	ProposedPluginStatusArchived  ProposedPluginStatus = "archived"
)

type ProposedPluginPricing string

const (
	ProposedPluginPricingFree       ProposedPluginPricing = "free"
	ProposedPluginPricingPerTx      ProposedPluginPricing = "per-tx"
	ProposedPluginPricingPerInstall ProposedPluginPricing = "per-install"
)

type ProposedPlugin struct {
	PluginID         string                 `json:"plugin_id"`
	PublicKey        string                 `json:"public_key"`
	Title            string                 `json:"title"`
	ShortDescription string                 `json:"short_description"`
	ServerEndpoint   string                 `json:"server_endpoint"`
	SupportedChains  []string               `json:"supported_chains"`
	PricingModel     *ProposedPluginPricing `json:"pricing_model"`
	ContactEmail     string                 `json:"contact_email"`
	Notes            string                 `json:"notes"`
	Status           ProposedPluginStatus   `json:"status"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
}

type ProposedPluginCreateParams struct {
	PluginID         string
	PublicKey        string
	Title            string
	ShortDescription string
	ServerEndpoint   string
	SupportedChains  []string
	PricingModel     *ProposedPluginPricing
	ContactEmail     string
	Notes            string
}

type ProposedPluginImage struct {
	ID                  uuid.UUID       `json:"id"`
	PluginID            string          `json:"plugin_id"`
	ImageType           PluginImageType `json:"image_type"`
	S3Path              string          `json:"s3_path"`
	ImageOrder          int             `json:"image_order"`
	UploadedByPublicKey string          `json:"-"`
	Visible             bool            `json:"-"`
	Deleted             bool            `json:"-"`
	ContentType         string          `json:"content_type"`
	Filename            string          `json:"filename"`
	CreatedAt           time.Time       `json:"created_at"`
	UpdatedAt           time.Time       `json:"updated_at"`
}

type ProposedPluginImageCreateParams struct {
	ID                  uuid.UUID
	PluginID            string
	ImageType           PluginImageType
	S3Path              string
	ImageOrder          int
	UploadedByPublicKey string
	ContentType         string
	Filename            string
}
