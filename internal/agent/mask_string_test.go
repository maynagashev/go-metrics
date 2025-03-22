//nolint:testpackage // использует внутреннее API агента для тестирования
package agent

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestMaskString проверяет функцию maskString.
func TestMaskString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Пустая строка",
			input:    "",
			expected: "<empty>",
		},
		{
			name:     "Короткая строка (меньше minMaskLength)",
			input:    "abcde",
			expected: "<empty>",
		},
		{
			name:     "Строка точно minMaskLength",
			input:    "abcdef",
			expected: "ab**ef",
		},
		{
			name:     "Длинная строка",
			input:    "abcdefghijklmnopqrstuvwxyz",
			expected: "ab**********************yz",
		},
		{
			name:     "Реальный ключ",
			input:    "-----BEGIN RSA PRIVATE KEY----- ... -----END RSA PRIVATE KEY-----",
			expected: "--*************************************************************--",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := maskString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
