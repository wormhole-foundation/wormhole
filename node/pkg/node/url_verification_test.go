package node

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateURL(t *testing.T) {
	tests := []struct {
		validSchemes []string
		urlStr       string
		expected     bool
	}{
		{[]string{"http", "https"}, "http://example.com", true},
		{[]string{"http", "https"}, "https://example.com", true},
		{[]string{"http", "https"}, "invalid-url", false},
		{[]string{"http", "https"}, "http://example.com:8080", true},
		{[]string{"http", "https"}, "https://example.com:8080", true},
		{[]string{"http", "https"}, "http://example.com:8080/path", true},
		{[]string{"http", "https"}, "https://example.com:8080/path", true},
		{[]string{"http", "https"}, "", false},
		{[]string{"http", "https"}, "ws://invalid-scheme.com:8080", false},
		{[]string{"http", "https"}, "wss://invalid-scheme.com:8080", false},
		{[]string{""}, "example.com:8080", true},
		{[]string{""}, "http://invalid-scheme:8080", false},
		{[]string{""}, "ws://invalid-scheme:8080", false},
		{[]string{""}, "170.0.0.1:8080", true},
	}

	for _, test := range tests {
		result := validateURL(test.urlStr, test.validSchemes)
		assert.Equal(t, test.expected, result)
	}
}
