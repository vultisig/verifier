package portal

import (
	"encoding/binary"
	"errors"
	"math"
)

var (
	ErrInvalidDimensions = errors.New("failed to parse image dimensions")
	ErrUnsupportedFormat = errors.New("unsupported image format")
)

// DetectContentType identifies image format by checking magic bytes at the start of the file.
func DetectContentType(data []byte) string {
	// PNG: 8-byte signature 89 50 4E 47 0D 0A 1A 0A
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

	// JPEG: starts with FF D8 (SOI marker)
	if len(data) >= 2 && data[0] == 0xFF && data[1] == 0xD8 {
		return "image/jpeg"
	}

	// WebP: RIFF container with "WEBP" format identifier at offset 8
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

// parsePNGDimensions reads dimensions from PNG IHDR chunk.
// PNG structure: 8-byte signature, then chunks. First chunk is always IHDR.
// IHDR layout at offset 16: 4 bytes width (big-endian), 4 bytes height (big-endian).
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

	w32 := binary.BigEndian.Uint32(data[16:20])
	h32 := binary.BigEndian.Uint32(data[20:24])

	if w32 == 0 || h32 == 0 || w32 > math.MaxInt32 || h32 > math.MaxInt32 {
		return 0, 0, ErrInvalidDimensions
	}

	return int(w32), int(h32), nil
}

// parseJPEGDimensions scans JPEG markers to find SOF (Start of Frame) which contains dimensions.
// JPEG is a sequence of segments, each starting with FF xx (marker), followed by 2-byte length.
// SOF markers (C0-CF except C4, C8, CC) contain: length, precision, height (2 bytes), width (2 bytes).
func parseJPEGDimensions(data []byte) (int, int, error) {
	// Verify SOI (Start of Image) marker
	if len(data) < 2 || data[0] != 0xFF || data[1] != 0xD8 {
		return 0, 0, ErrInvalidDimensions
	}

	i := 2
	for i < len(data)-1 {
		// Find next marker (FF followed by non-FF byte)
		if data[i] != 0xFF {
			i++
			continue
		}

		// Skip padding FF bytes
		for i < len(data) && data[i] == 0xFF {
			i++
		}
		if i >= len(data) {
			break
		}

		marker := data[i]
		i++

		// EOI (End of Image) - stop scanning
		if marker == 0xD9 {
			break
		}

		// Standalone markers without length field: SOI (D8), TEM (01), RST0-RST7 (D0-D7)
		if marker == 0xD8 || marker == 0x01 || (marker >= 0xD0 && marker <= 0xD7) {
			continue
		}

		// Read segment length (includes the 2 length bytes, excludes marker)
		if i+2 > len(data) {
			break
		}
		segmentLen := int(binary.BigEndian.Uint16(data[i : i+2]))
		if segmentLen < 2 {
			break
		}

		// SOF markers contain dimensions at offset +3 (height) and +5 (width) from length field
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

// isSOFMarker returns true if marker is a Start of Frame marker containing image dimensions.
// SOF markers are C0-CF, excluding C4 (DHT), C8 (reserved), CC (DAC).
func isSOFMarker(marker byte) bool {
	if marker >= 0xC0 && marker <= 0xCF {
		if marker == 0xC4 || marker == 0xC8 || marker == 0xCC {
			return false
		}
		return true
	}
	return false
}

// parseWebPDimensions handles WebP's three encoding formats: VP8 (lossy), VP8L (lossless), VP8X (extended).
// WebP uses RIFF container: "RIFF" + file size + "WEBP" + chunks.
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

	// First chunk after WEBP header determines the format
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

// parseVP8Dimensions extracts dimensions from VP8 lossy bitstream.
// VP8 chunk: "VP8 " + 4-byte size + frame data.
// Frame starts with 3-byte frame tag, then sync code 9D 01 2A, then 2 bytes width, 2 bytes height.
// Width/height are 14-bit values (lower 14 bits, upper 2 bits are scale factor).
func parseVP8Dimensions(data []byte) (int, int, error) {
	if len(data) < 18 {
		return 0, 0, ErrInvalidDimensions
	}

	// Frame tag at offset 8 (after "VP8 " + 4-byte size)
	frameTag := data[8:11]
	// Bit 0 of frame tag must be 0 (keyframe)
	if frameTag[0]&0x01 != 0 {
		return 0, 0, ErrInvalidDimensions
	}

	// VP8 sync code: 9D 01 2A
	if data[11] != 0x9D || data[12] != 0x01 || data[13] != 0x2A {
		return 0, 0, ErrInvalidDimensions
	}

	// Width and height are 14-bit little-endian values
	width := int(data[14]) | (int(data[15]&0x3F) << 8)
	height := int(data[16]) | (int(data[17]&0x3F) << 8)

	if width <= 0 || height <= 0 {
		return 0, 0, ErrInvalidDimensions
	}

	return width, height, nil
}

// parseVP8LDimensions extracts dimensions from VP8L lossless bitstream.
// VP8L chunk: "VP8L" + 4-byte size + signature byte (0x2F) + 4 bytes containing width/height.
// Dimensions are packed: bits 0-13 = width-1, bits 14-27 = height-1.
func parseVP8LDimensions(data []byte) (int, int, error) {
	if len(data) < 13 {
		return 0, 0, ErrInvalidDimensions
	}

	// VP8L signature byte at offset 8
	if data[8] != 0x2F {
		return 0, 0, ErrInvalidDimensions
	}

	// Read 4 bytes as little-endian uint32, extract packed dimensions
	bits := uint32(data[9]) | uint32(data[10])<<8 | uint32(data[11])<<16 | uint32(data[12])<<24
	width := int((bits & 0x3FFF) + 1)
	height := int(((bits >> 14) & 0x3FFF) + 1)

	if width <= 0 || height <= 0 {
		return 0, 0, ErrInvalidDimensions
	}

	return width, height, nil
}

// parseVP8XDimensions extracts dimensions from VP8X extended format header.
// VP8X chunk: "VP8X" + 4-byte size + 4-byte flags + 3-byte width-1 + 3-byte height-1.
// This format supports alpha, animation, and other extensions.
func parseVP8XDimensions(data []byte) (int, int, error) {
	if len(data) < 18 {
		return 0, 0, ErrInvalidDimensions
	}

	// Canvas dimensions are 24-bit little-endian values at offsets 12 and 15
	width := int(data[12]) | int(data[13])<<8 | int(data[14])<<16
	width++

	height := int(data[15]) | int(data[16])<<8 | int(data[17])<<16
	height++

	if width <= 0 || height <= 0 {
		return 0, 0, ErrInvalidDimensions
	}

	return width, height, nil
}
