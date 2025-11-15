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

	tests := []test{
		{input: "MainNet", output: MainNet, err: false},
		{input: "Prod", output: MainNet, err: false},

		{input: "TestNet", output: TestNet, err: false},
		{input: "test", output: TestNet, err: false},

		{input: "UnsafeDevNet", output: UnsafeDevNet, err: false},
		{input: "devnet", output: UnsafeDevNet, err: false},
		{input: "dev", output: UnsafeDevNet, err: false},

		{input: "GoTest", output: GoTest, err: false},
		{input: "unit-test", output: GoTest, err: false},

		{input: "AccountantMock", output: AccountantMock, err: false},
		{input: "accountant-mock", output: AccountantMock, err: false},

		{input: "junk", output: UnsafeDevNet, err: true},
		{input: "", output: UnsafeDevNet, err: true},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			output, err := ParseEnvironment(tc.input)
			if err != nil {
				if tc.err == false {
					assert.NoError(t, err)
				}
			} else if tc.err {
				assert.Error(t, err)
			} else {
				assert.Equal(t, tc.output, output)
			}
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
