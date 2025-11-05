package validator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsNumericRegex(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "only digits",
			input:    "1234567890",
			expected: true,
		},
		{
			name:     "single digit",
			input:    "5",
			expected: true,
		},
		{
			name:     "empty string",
			input:    "",
			expected: false,
		},
		{
			name:     "letters only",
			input:    "abc",
			expected: false,
		},
		{
			name:     "mixed letters and digits",
			input:    "123abc",
			expected: false,
		},
		{
			name:     "digits with spaces",
			input:    "123 456",
			expected: false,
		},
		{
			name:     "digits with special characters",
			input:    "123-456",
			expected: false,
		},
		{
			name:     "decimal number",
			input:    "123.45",
			expected: false,
		},
		{
			name:     "negative number",
			input:    "-123",
			expected: false,
		},
		{
			name:     "whitespace only",
			input:    "   ",
			expected: false,
		},
		{
			name:     "digits with space",
			input:    " 123 ",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsNumericRegex(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
