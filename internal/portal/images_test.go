package portal

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "normal filename",
			input:    "photo.jpg",
			expected: "photo.jpg",
		},
		{
			name:     "filename with spaces",
			input:    "my photo.jpg",
			expected: "my photo.jpg",
		},
		{
			name:     "path traversal attack",
			input:    "../../../etc/passwd",
			expected: "passwd",
		},
		{
			name:     "windows path traversal",
			input:    "..\\..\\..\\windows\\system32\\config",
			expected: "config",
		},
		{
			name:     "absolute path unix",
			input:    "/etc/passwd",
			expected: "passwd",
		},
		{
			name:     "absolute path windows",
			input:    "C:\\Users\\Admin\\file.txt",
			expected: "file.txt",
		},
		{
			name:     "control characters",
			input:    "file\x00name\x1f.jpg",
			expected: "filename.jpg",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "file",
		},
		{
			name:     "just dots",
			input:    ".",
			expected: "file",
		},
		{
			name:     "double dots",
			input:    "..",
			expected: "file",
		},
		{
			name:     "hidden file",
			input:    ".htaccess",
			expected: ".htaccess",
		},
		{
			name:     "unicode filename",
			input:    "фото.jpg",
			expected: "фото.jpg",
		},
		{
			name:     "emoji filename",
			input:    "image🎉.png",
			expected: "image🎉.png",
		},
		{
			name:     "long filename truncated",
			input:    strings.Repeat("a", 300) + ".jpg",
			expected: strings.Repeat("a", 255),
		},
		{
			name:     "exactly 255 chars",
			input:    strings.Repeat("b", 255),
			expected: strings.Repeat("b", 255),
		},
		{
			name:     "256 chars truncated",
			input:    strings.Repeat("c", 256),
			expected: strings.Repeat("c", 255),
		},
		{
			name:     "multiple slashes",
			input:    "path/to/some/file.jpg",
			expected: "file.jpg",
		},
		{
			name:     "tab character stripped",
			input:    "file\tname.jpg",
			expected: "filename.jpg",
		},
		{
			name:     "newline stripped",
			input:    "file\nname.jpg",
			expected: "filename.jpg",
		},
		{
			name:     "delete character removed",
			input:    "file\x7fname.jpg",
			expected: "filename.jpg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeFilename(tt.input)
			assert.Equal(t, tt.expected, result)
			assert.LessOrEqual(t, len(result), 255, "filename should not exceed 255 characters")
		})
	}
}

func TestContentTypeToExt(t *testing.T) {
	tests := []struct {
		contentType string
		expected    string
	}{
		{"image/jpeg", ".jpg"},
		{"image/png", ".png"},
		{"image/webp", ".webp"},
		{"image/gif", ""},
		{"text/plain", ""},
		{"", ""},
		{"application/octet-stream", ""},
	}

	for _, tt := range tests {
		t.Run(tt.contentType, func(t *testing.T) {
			result := contentTypeToExt(tt.contentType)
			assert.Equal(t, tt.expected, result)
		})
	}
}
