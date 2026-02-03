package portal

import (
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectContentType(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected string
	}{
		{
			name:     "PNG signature",
			data:     []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00},
			expected: "image/png",
		},
		{
			name:     "JPEG signature",
			data:     []byte{0xFF, 0xD8, 0xFF, 0xE0},
			expected: "image/jpeg",
		},
		{
			name:     "WebP signature",
			data:     []byte{'R', 'I', 'F', 'F', 0x00, 0x00, 0x00, 0x00, 'W', 'E', 'B', 'P'},
			expected: "image/webp",
		},
		{
			name:     "unknown format",
			data:     []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07},
			expected: "",
		},
		{
			name:     "empty data",
			data:     []byte{},
			expected: "",
		},
		{
			name:     "truncated PNG",
			data:     []byte{0x89, 0x50, 0x4E},
			expected: "",
		},
		{
			name:     "truncated JPEG",
			data:     []byte{0xFF},
			expected: "",
		},
		{
			name:     "partial WebP",
			data:     []byte{'R', 'I', 'F', 'F', 0x00, 0x00, 0x00, 0x00, 'W', 'E', 'B'},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectContentType(tt.data)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParsePNGDimensions(t *testing.T) {
	tests := []struct {
		name        string
		width       int
		height      int
		expectError bool
	}{
		{name: "512x512", width: 512, height: 512, expectError: false},
		{name: "1920x1080", width: 1920, height: 1080, expectError: false},
		{name: "1x1", width: 1, height: 1, expectError: false},
		{name: "large dimensions", width: 4096, height: 4096, expectError: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := buildPNG(tt.width, tt.height)
			w, h, err := ParseImageDimensions(data, "image/png")

			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.width, w)
				assert.Equal(t, tt.height, h)
			}
		})
	}
}

func TestParsePNGDimensions_Invalid(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{name: "empty data", data: []byte{}},
		{name: "too short", data: []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}},
		{name: "wrong signature", data: make([]byte, 24)},
		{name: "truncated IHDR", data: []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := ParseImageDimensions(tt.data, "image/png")
			assert.Error(t, err)
		})
	}
}

func TestParseJPEGDimensions(t *testing.T) {
	tests := []struct {
		name   string
		width  int
		height int
	}{
		{name: "800x600", width: 800, height: 600},
		{name: "1920x1080", width: 1920, height: 1080},
		{name: "512x512", width: 512, height: 512},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := buildJPEG(tt.width, tt.height)
			w, h, err := ParseImageDimensions(data, "image/jpeg")

			require.NoError(t, err)
			assert.Equal(t, tt.width, w)
			assert.Equal(t, tt.height, h)
		})
	}
}

func TestParseJPEGDimensions_WithLargeAPP(t *testing.T) {
	data := buildJPEGWithLargeAPP(800, 600, 70000)
	w, h, err := ParseImageDimensions(data, "image/jpeg")

	require.NoError(t, err)
	assert.Equal(t, 800, w)
	assert.Equal(t, 600, h)
}

func TestParseJPEGDimensions_Invalid(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{name: "empty data", data: []byte{}},
		{name: "wrong signature", data: []byte{0x00, 0x00}},
		{name: "only SOI marker", data: []byte{0xFF, 0xD8}},
		{name: "truncated", data: []byte{0xFF, 0xD8, 0xFF, 0xC0}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := ParseImageDimensions(tt.data, "image/jpeg")
			assert.Error(t, err)
		})
	}
}

func TestParseWebPDimensions_VP8(t *testing.T) {
	data := buildWebPVP8(640, 480)
	w, h, err := ParseImageDimensions(data, "image/webp")

	require.NoError(t, err)
	assert.Equal(t, 640, w)
	assert.Equal(t, 480, h)
}

func TestParseWebPDimensions_VP8L(t *testing.T) {
	data := buildWebPVP8L(512, 512)
	w, h, err := ParseImageDimensions(data, "image/webp")

	require.NoError(t, err)
	assert.Equal(t, 512, w)
	assert.Equal(t, 512, h)
}

func TestParseWebPDimensions_VP8X(t *testing.T) {
	data := buildWebPVP8X(1920, 1080)
	w, h, err := ParseImageDimensions(data, "image/webp")

	require.NoError(t, err)
	assert.Equal(t, 1920, w)
	assert.Equal(t, 1080, h)
}

func TestParseWebPDimensions_Invalid(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{name: "empty data", data: []byte{}},
		{name: "too short", data: []byte{'R', 'I', 'F', 'F'}},
		{name: "wrong RIFF", data: []byte{'X', 'I', 'F', 'F', 0, 0, 0, 0, 'W', 'E', 'B', 'P'}},
		{name: "wrong WEBP", data: []byte{'R', 'I', 'F', 'F', 0, 0, 0, 0, 'X', 'E', 'B', 'P'}},
		{name: "unknown chunk", data: []byte{'R', 'I', 'F', 'F', 0, 0, 0, 0, 'W', 'E', 'B', 'P', 'X', 'X', 'X', 'X'}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := ParseImageDimensions(tt.data, "image/webp")
			assert.Error(t, err)
		})
	}
}

func TestParseImageDimensions_UnsupportedFormat(t *testing.T) {
	_, _, err := ParseImageDimensions([]byte{0x00}, "image/gif")
	assert.ErrorIs(t, err, ErrUnsupportedFormat)
}

func buildPNG(width, height int) []byte {
	data := make([]byte, 33)
	copy(data[0:8], []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A})
	binary.BigEndian.PutUint32(data[8:12], 13)
	copy(data[12:16], []byte{'I', 'H', 'D', 'R'})
	binary.BigEndian.PutUint32(data[16:20], uint32(width))
	binary.BigEndian.PutUint32(data[20:24], uint32(height))
	data[24] = 8
	data[25] = 6
	return data
}

func buildJPEG(width, height int) []byte {
	data := make([]byte, 20)
	data[0] = 0xFF
	data[1] = 0xD8
	data[2] = 0xFF
	data[3] = 0xC0
	binary.BigEndian.PutUint16(data[4:6], 11)
	data[6] = 8
	binary.BigEndian.PutUint16(data[7:9], uint16(height))
	binary.BigEndian.PutUint16(data[9:11], uint16(width))
	return data
}

func buildJPEGWithLargeAPP(width, height, appSize int) []byte {
	data := make([]byte, 4+appSize+13)
	data[0] = 0xFF
	data[1] = 0xD8
	data[2] = 0xFF
	data[3] = 0xE1
	binary.BigEndian.PutUint16(data[4:6], uint16(appSize))
	offset := 4 + appSize
	data[offset] = 0xFF
	data[offset+1] = 0xC0
	binary.BigEndian.PutUint16(data[offset+2:offset+4], 11)
	data[offset+4] = 8
	binary.BigEndian.PutUint16(data[offset+5:offset+7], uint16(height))
	binary.BigEndian.PutUint16(data[offset+7:offset+9], uint16(width))
	return data
}

func buildWebPVP8(width, height int) []byte {
	data := make([]byte, 30)
	copy(data[0:4], "RIFF")
	binary.LittleEndian.PutUint32(data[4:8], 22)
	copy(data[8:12], "WEBP")
	copy(data[12:16], "VP8 ")
	binary.LittleEndian.PutUint32(data[16:20], 10)
	data[20] = 0x00
	data[21] = 0x00
	data[22] = 0x00
	data[23] = 0x9D
	data[24] = 0x01
	data[25] = 0x2A
	data[26] = byte(width & 0xFF)
	data[27] = byte((width >> 8) & 0x3F)
	data[28] = byte(height & 0xFF)
	data[29] = byte((height >> 8) & 0x3F)
	return data
}

func buildWebPVP8L(width, height int) []byte {
	data := make([]byte, 25)
	copy(data[0:4], "RIFF")
	binary.LittleEndian.PutUint32(data[4:8], 17)
	copy(data[8:12], "WEBP")
	copy(data[12:16], "VP8L")
	binary.LittleEndian.PutUint32(data[16:20], 5)
	data[20] = 0x2F

	w := width - 1
	h := height - 1
	bits := uint32(w&0x3FFF) | uint32(h&0x3FFF)<<14
	data[21] = byte(bits)
	data[22] = byte(bits >> 8)
	data[23] = byte(bits >> 16)
	data[24] = byte(bits >> 24)
	return data
}

func buildWebPVP8X(width, height int) []byte {
	data := make([]byte, 30)
	copy(data[0:4], "RIFF")
	binary.LittleEndian.PutUint32(data[4:8], 22)
	copy(data[8:12], "WEBP")
	copy(data[12:16], "VP8X")
	binary.LittleEndian.PutUint32(data[16:20], 10)
	data[20] = 0x00
	data[21] = 0x00
	data[22] = 0x00
	data[23] = 0x00

	w := width - 1
	h := height - 1
	data[24] = byte(w)
	data[25] = byte(w >> 8)
	data[26] = byte(w >> 16)
	data[27] = byte(h)
	data[28] = byte(h >> 8)
	data[29] = byte(h >> 16)
	return data
}
