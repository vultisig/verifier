package portal

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"

	_ "golang.org/x/image/webp"
)

var (
	ErrInvalidImage      = errors.New("invalid image")
	ErrInvalidDimensions = errors.New("invalid image dimensions")
	ErrUnsupportedFormat = errors.New("unsupported image format")
)

type ImageInfo struct {
	Width  int
	Height int
	MIME   string
}

func ParseImageInfo(data []byte) (ImageInfo, error) {
	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return ImageInfo{}, fmt.Errorf("%w: %w", ErrInvalidImage, err)
	}

	var mime string
	switch format {
	case "png":
		mime = "image/png"
	case "jpeg":
		mime = "image/jpeg"
	case "webp":
		mime = "image/webp"
	default:
		return ImageInfo{}, fmt.Errorf("%w: %s", ErrUnsupportedFormat, format)
	}

	if cfg.Width <= 0 || cfg.Height <= 0 {
		return ImageInfo{}, ErrInvalidDimensions
	}

	return ImageInfo{Width: cfg.Width, Height: cfg.Height, MIME: mime}, nil
}
