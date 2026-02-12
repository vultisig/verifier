package portal

import "time"

type ImageData struct {
	Type        string `json:"type" validate:"required,oneof=logo banner thumbnail media"`
	Data        string `json:"data" validate:"required"`
	ContentType string `json:"content_type" validate:"required,oneof=image/png image/jpeg image/webp"`
	Filename    string `json:"filename" validate:"omitempty,max=255"`
}

type CreateProposedPluginRequest struct {
	PluginID         string      `json:"plugin_id" validate:"required,max=64"`
	Title            string      `json:"title" validate:"required,max=255"`
	ShortDescription string      `json:"short_description" validate:"omitempty,max=500"`
	ServerEndpoint   string      `json:"server_endpoint" validate:"required"`
	SupportedChains  []string    `json:"supported_chains" validate:"omitempty,dive,max=32"`
	PricingModel     string      `json:"pricing_model" validate:"omitempty,oneof=free per-tx per-install"`
	ContactEmail     string      `json:"contact_email" validate:"required,email,max=255"`
	Notes            string      `json:"notes" validate:"omitempty,max=2000"`
	Images           []ImageData `json:"images" validate:"required,min=1,max=9,dive"`
}

type ProposedPluginImageResponse struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	URL         string `json:"url"`
	ContentType string `json:"content_type"`
	Filename    string `json:"filename"`
	ImageOrder  int    `json:"image_order"`
}

type ProposedPluginResponse struct {
	PluginID         string                        `json:"plugin_id"`
	Title            string                        `json:"title"`
	ShortDescription string                        `json:"short_description"`
	ServerEndpoint   string                        `json:"server_endpoint"`
	SupportedChains  []string                      `json:"supported_chains"`
	PricingModel     *string                       `json:"pricing_model,omitempty"`
	ContactEmail     string                        `json:"contact_email"`
	Notes            string                        `json:"notes"`
	Status           string                        `json:"status"`
	Images           []ProposedPluginImageResponse `json:"images"`
	CreatedAt        time.Time                     `json:"created_at"`
	UpdatedAt        time.Time                     `json:"updated_at"`
}

type ValidatePluginIDResponse struct {
	Available bool `json:"available"`
}
