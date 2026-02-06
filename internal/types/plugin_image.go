package types

import (
	"time"

	"github.com/google/uuid"
	"github.com/vultisig/verifier/types"
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
	PluginID            types.PluginID  `json:"plugin_id"`
	ImageType           PluginImageType `json:"image_type"`
	S3Path              string          `json:"s3_path"`
	ImageOrder          int             `json:"image_order"`
	UploadedByPublicKey string          `json:"-"`
	Visible             bool            `json:"-"`
	Deleted             bool            `json:"-"`
	CreatedAt           time.Time       `json:"created_at"`
	UpdatedAt           time.Time       `json:"updated_at"`
}

type PluginImageCreateParams struct {
	PluginID            types.PluginID
	ImageType           PluginImageType
	S3Path              string
	ImageOrder          int
	UploadedByPublicKey string
}
