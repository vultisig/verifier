package portal

import (
	"encoding/binary"
	"errors"
)

var (
	ErrInvalidDimensions = errors.New("failed to parse image dimensions")
	ErrUnsupportedFormat = errors.New("unsupported image format")
)

func DetectContentType(data []byte) string {
	if len(data) >= 8 {
		pngSignature := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
		isPNG := true
		for i := 0; i < 8; i++ {
			if data[i] != pngSignature[i] {
				isPNG = false
				break
			}
		}
		if isPNG {
			return "image/png"
		}
	}

	if len(data) >= 2 && data[0] == 0xFF && data[1] == 0xD8 {
		return "image/jpeg"
	}

	if len(data) >= 12 && string(data[0:4]) == "RIFF" && string(data[8:12]) == "WEBP" {
		return "image/webp"
	}

	return ""
}

func ParseImageDimensions(data []byte, contentType string) (width, height int, err error) {
	switch contentType {
	case "image/png":
		return parsePNGDimensions(data)
	case "image/jpeg":
		return parseJPEGDimensions(data)
	case "image/webp":
		return parseWebPDimensions(data)
	default:
		return 0, 0, ErrUnsupportedFormat
	}
}

func parsePNGDimensions(data []byte) (int, int, error) {
	if len(data) < 24 {
		return 0, 0, ErrInvalidDimensions
	}

	pngSignature := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	for i := 0; i < 8; i++ {
		if data[i] != pngSignature[i] {
			return 0, 0, ErrInvalidDimensions
		}
	}

	width := int(binary.BigEndian.Uint32(data[16:20]))
	height := int(binary.BigEndian.Uint32(data[20:24]))

	if width <= 0 || height <= 0 {
		return 0, 0, ErrInvalidDimensions
	}

	return width, height, nil
}

func parseJPEGDimensions(data []byte) (int, int, error) {
	if len(data) < 2 || data[0] != 0xFF || data[1] != 0xD8 {
		return 0, 0, ErrInvalidDimensions
	}

	i := 2
	for i < len(data)-1 {
		if data[i] != 0xFF {
			i++
			continue
		}

		for i < len(data) && data[i] == 0xFF {
			i++
		}
		if i >= len(data) {
			break
		}

		marker := data[i]
		i++

		if marker == 0xD9 {
			break
		}

		if marker == 0xD8 || marker == 0x01 || (marker >= 0xD0 && marker <= 0xD7) {
			continue
		}

		if i+2 > len(data) {
			break
		}
		segmentLen := int(binary.BigEndian.Uint16(data[i : i+2]))
		if segmentLen < 2 {
			break
		}

		if isSOFMarker(marker) {
			if i+7 > len(data) {
				return 0, 0, ErrInvalidDimensions
			}
			height := int(binary.BigEndian.Uint16(data[i+3 : i+5]))
			width := int(binary.BigEndian.Uint16(data[i+5 : i+7]))
			if width <= 0 || height <= 0 {
				return 0, 0, ErrInvalidDimensions
			}
			return width, height, nil
		}

		i += segmentLen
	}

	return 0, 0, ErrInvalidDimensions
}

func isSOFMarker(marker byte) bool {
	if marker >= 0xC0 && marker <= 0xCF {
		if marker == 0xC4 || marker == 0xC8 || marker == 0xCC {
			return false
		}
		return true
	}
	return false
}

func parseWebPDimensions(data []byte) (int, int, error) {
	if len(data) < 12 {
		return 0, 0, ErrInvalidDimensions
	}

	if string(data[0:4]) != "RIFF" || string(data[8:12]) != "WEBP" {
		return 0, 0, ErrInvalidDimensions
	}

	if len(data) < 16 {
		return 0, 0, ErrInvalidDimensions
	}

	chunkType := string(data[12:16])

	switch chunkType {
	case "VP8 ":
		return parseVP8Dimensions(data[12:])
	case "VP8L":
		return parseVP8LDimensions(data[12:])
	case "VP8X":
		return parseVP8XDimensions(data[12:])
	default:
		return 0, 0, ErrInvalidDimensions
	}
}

func parseVP8Dimensions(data []byte) (int, int, error) {
	if len(data) < 18 {
		return 0, 0, ErrInvalidDimensions
	}

	frameTag := data[8:11]
	if frameTag[0]&0x01 != 0 {
		return 0, 0, ErrInvalidDimensions
	}

	if data[11] != 0x9D || data[12] != 0x01 || data[13] != 0x2A {
		return 0, 0, ErrInvalidDimensions
	}

	width := int(data[14]) | (int(data[15]&0x3F) << 8)
	height := int(data[16]) | (int(data[17]&0x3F) << 8)

	if width <= 0 || height <= 0 {
		return 0, 0, ErrInvalidDimensions
	}

	return width, height, nil
}

func parseVP8LDimensions(data []byte) (int, int, error) {
	if len(data) < 13 {
		return 0, 0, ErrInvalidDimensions
	}

	if data[8] != 0x2F {
		return 0, 0, ErrInvalidDimensions
	}

	bits := uint32(data[9]) | uint32(data[10])<<8 | uint32(data[11])<<16 | uint32(data[12])<<24
	width := int((bits & 0x3FFF) + 1)
	height := int(((bits >> 14) & 0x3FFF) + 1)

	if width <= 0 || height <= 0 {
		return 0, 0, ErrInvalidDimensions
	}

	return width, height, nil
}

func parseVP8XDimensions(data []byte) (int, int, error) {
	if len(data) < 18 {
		return 0, 0, ErrInvalidDimensions
	}

	width := int(data[12]) | int(data[13])<<8 | int(data[14])<<16
	width++

	height := int(data[15]) | int(data[16])<<8 | int(data[17])<<16
	height++

	if width <= 0 || height <= 0 {
		return 0, 0, ErrInvalidDimensions
	}

	return width, height, nil
}
