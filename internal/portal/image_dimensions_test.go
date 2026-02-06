package portal

import (
	"bytes"
	"errors"
	"image"
	"image/jpeg"
	"image/png"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParsePNGDimensions(t *testing.T) {
	tests := []struct {
		name   string
		width  int
		height int
	}{
		{name: "512x512", width: 512, height: 512},
		{name: "1920x1080", width: 1920, height: 1080},
		{name: "1x1", width: 1, height: 1},
		{name: "100x200", width: 100, height: 200},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := createPNG(tt.width, tt.height)
			info, err := ParseImageInfo(data)

			require.NoError(t, err)
			assert.Equal(t, tt.width, info.Width)
			assert.Equal(t, tt.height, info.Height)
			assert.Equal(t, "image/png", info.MIME)
		})
	}
}

func TestParsePNGDimensions_Invalid(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{name: "empty data", data: []byte{}},
		{name: "truncated", data: []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}},
		{name: "wrong signature", data: make([]byte, 100)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseImageInfo(tt.data)
			assert.Error(t, err)
			assert.True(t, errors.Is(err, ErrInvalidImage) || errors.Is(err, ErrUnsupportedFormat))
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
			data := createJPEG(tt.width, tt.height)
			info, err := ParseImageInfo(data)

			require.NoError(t, err)
			assert.Equal(t, tt.width, info.Width)
			assert.Equal(t, tt.height, info.Height)
			assert.Equal(t, "image/jpeg", info.MIME)
		})
	}
}

func TestParseJPEGDimensions_Invalid(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{name: "empty data", data: []byte{}},
		{name: "wrong signature", data: []byte{0x00, 0x00}},
		{name: "truncated", data: []byte{0xFF, 0xD8, 0xFF, 0xC0}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseImageInfo(tt.data)
			assert.Error(t, err)
		})
	}
}

func TestParseWebPDimensions(t *testing.T) {
	data := createWebPLossless(640, 480)
	info, err := ParseImageInfo(data)

	require.NoError(t, err)
	assert.Equal(t, 640, info.Width)
	assert.Equal(t, 480, info.Height)
	assert.Equal(t, "image/webp", info.MIME)
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseImageInfo(tt.data)
			assert.Error(t, err)
		})
	}
}

func TestParseImageInfo_InvalidImage(t *testing.T) {
	_, err := ParseImageInfo([]byte{0x00, 0x01, 0x02, 0x03})
	assert.ErrorIs(t, err, ErrInvalidImage)
}

func createPNG(width, height int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	var buf bytes.Buffer
	png.Encode(&buf, img)
	return buf.Bytes()
}

func createJPEG(width, height int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	var buf bytes.Buffer
	jpeg.Encode(&buf, img, &jpeg.Options{Quality: 1})
	return buf.Bytes()
}

func createWebPLossless(width, height int) []byte {
	w := width - 1
	h := height - 1
	bits := uint32(w&0x3FFF) | uint32(h&0x3FFF)<<14

	chunkSize := uint32(5)
	fileSize := uint32(4 + 8 + chunkSize)

	data := make([]byte, 12+8+chunkSize)
	copy(data[0:4], "RIFF")
	data[4] = byte(fileSize)
	data[5] = byte(fileSize >> 8)
	data[6] = byte(fileSize >> 16)
	data[7] = byte(fileSize >> 24)
	copy(data[8:12], "WEBP")
	copy(data[12:16], "VP8L")
	data[16] = byte(chunkSize)
	data[17] = byte(chunkSize >> 8)
	data[18] = byte(chunkSize >> 16)
	data[19] = byte(chunkSize >> 24)
	data[20] = 0x2F
	data[21] = byte(bits)
	data[22] = byte(bits >> 8)
	data[23] = byte(bits >> 16)
	data[24] = byte(bits >> 24)

	return data
}
