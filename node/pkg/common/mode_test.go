package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wormhole-foundation/wormhole/sdk"
)

func TestParseEnvironment(t *testing.T) {
	tests := []struct {
		input    string
		expected Environment
		wantErr  bool
	}{
		{"prod", MainNet, false},
		{"mainnet", MainNet, false},
		{"test", TestNet, false},
		{"testnet", TestNet, false},
		{"dev", UnsafeDevNet, false},
		{"devnet", UnsafeDevNet, false},
		{"unsafedevnet", UnsafeDevNet, false},
		{"unit-test", GoTest, false},
		{"gotest", GoTest, false},
		{"accountant-mock", AccountantMock, false},
		{"accountantmock", AccountantMock, false},
		{"invalid", UnsafeDevNet, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := ParseEnvironment(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEnvironment_ToSDK(t *testing.T) {
	tests := []struct {
		name     string
		env      Environment
		expected sdk.Environment
	}{
		{
			name:     "MainNet",
			env:      MainNet,
			expected: sdk.EnvMainNet,
		},
		{
			name:     "TestNet",
			env:      TestNet,
			expected: sdk.EnvTestNet,
		},
		{
			name:     "UnsafeDevNet",
			env:      UnsafeDevNet,
			expected: sdk.EnvDevNet,
		},
		{
			name:     "GoTest",
			env:      GoTest,
			expected: sdk.EnvGoTest,
		},
		{
			name:     "AccountantMock",
			env:      AccountantMock,
			expected: sdk.EnvAccountantMock,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.env.ToSDK()
			assert.Equal(t, tt.expected, result)
		})
	}
}
