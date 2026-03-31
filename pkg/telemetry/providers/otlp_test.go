package providers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseOTLPHeaders(t *testing.T) {
	tests := []struct {
		name     string
		in       []string
		expected map[string]string
	}{
		{
			name:     "empty",
			in:       nil,
			expected: map[string]string{},
		},
		{
			name: "trims spaces",
			in:   []string{"Authorization=Bearer abc", " X-Key = value "},
			expected: map[string]string{
				"Authorization": "Bearer abc",
				"X-Key":         "value",
			},
		},
		{
			name: "ignores malformed pairs",
			in:   []string{"invalid", "k=v", "another-invalid"},
			expected: map[string]string{
				"k": "v",
			},
		},
		{
			name: "last duplicate key wins",
			in:   []string{"k=v1", "k=v2"},
			expected: map[string]string{
				"k": "v2",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseOTLPHeaders(tt.in)
			assert.Equal(t, tt.expected, got)
		})
	}
}
