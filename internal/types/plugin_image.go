package types

import (
	"time"

	"github.com/google/uuid"
)

type PluginImageType string

const (
	PluginImageTypeLogo      PluginImageType = "logo"
	PluginImageTypeBanner    PluginImageType = "banner"
	PluginImageTypeThumbnail PluginImageType = "thumbnail"
	PluginImageTypeMedia     PluginImageType = "media"
)

type PluginImageRecord struct {
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

type PluginImageCreateParams struct {
	ID                  uuid.UUID
	PluginID            string
	ImageType           PluginImageType
	S3Path              string
	ImageOrder          int
	UploadedByPublicKey string
	ContentType         string
	Filename            string
	Visible             bool
}

var AllowedContentTypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/webp": true,
}

type ImageConstraint struct {
	MaxWidth  int
	MaxHeight int
}

var ImageTypeConstraints = map[PluginImageType]ImageConstraint{
	PluginImageTypeLogo:      {MaxWidth: 512, MaxHeight: 512},
	PluginImageTypeBanner:    {MaxWidth: 1920, MaxHeight: 1080},
	PluginImageTypeThumbnail: {MaxWidth: 800, MaxHeight: 600},
	PluginImageTypeMedia:     {MaxWidth: 1920, MaxHeight: 1080},
}
