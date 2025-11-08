package validator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsValidLuhn(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "valid number with spaces",
			input:    "4532 0151 1283 0366",
			expected: true,
		},
		{
			name:     "invalid number with spaces",
			input:    "4532  0151  1283  0367",
			expected: false,
		},
		{
			name:     "empty string",
			input:    "",
			expected: false,
		},
		{
			name:     "single digit",
			input:    "1",
			expected: false,
		},
		{
			name:     "contains letters",
			input:    "4532a015112830366",
			expected: false,
		},
		{
			name:     "contains special characters",
			input:    "4532-0151-1283-0366",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidLuhn(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
