package types

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

var testToken [32]byte = [32]byte{0x16, 0x58, 0x09, 0x73, 0x92, 0x40, 0xa0, 0xac, 0x03, 0xb9, 0x84, 0x40, 0xfe, 0x89, 0x85, 0x54, 0x8e, 0x3a, 0xa6, 0x83, 0xcd, 0x0d, 0x4d, 0x9d, 0xf5, 0xb5, 0x65, 0x96, 0x69, 0xfa, 0xa3, 0x00}
var uworm [32]byte = [32]byte{0x16, 0x58, 0x09, 0x73, 0x92, 0x40, 0xa0, 0xac, 0x03, 0xb9, 0x84, 0x40, 0xfe, 0x89, 0x85, 0x54, 0x8e, 0x3a, 0xa6, 0x83, 0xcd, 0x0d, 0x4d, 0x9d, 0xf5, 0xb5, 0x65, 0x96, 0x69, 0xfa, 0xa3, 0x01}

func TestGetWrappedTokenIdentifier(t *testing.T) {
	tests := []struct {
		name         string
		tokenChain   uint16
		tokenAddress [32]byte
		result       string
	}{
		{
			name:         "Wrapped token from Solana",
			tokenChain:   1,
			tokenAddress: testToken,
			result:       "wh/00001/165809739240a0ac03b98440fe8985548e3aa683cd0d4d9df5b5659669faa300",
		},
		{
			name:         "uworm token (from Solana)",
			tokenChain:   1,
			tokenAddress: uworm,
			result:       "uworm",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetWrappedCoinIdentifier(tt.tokenChain, tt.tokenAddress)
			require.EqualValues(t, tt.result, result)
		})
	}
}

func TestGetWrappedCoinMeta(t *testing.T) {
	tests := []struct {
		name         string
		identifier   string
		tokenChain   uint16
		tokenAddress [32]byte
		wrapped      bool
	}{
		{
			name:         "Wrapped token from Solana",
			identifier:   "wh/00001/165809739240a0ac03b98440fe8985548e3aa683cd0d4d9df5b5659669faa300",
			tokenChain:   1,
			tokenAddress: testToken,
			wrapped:      true,
		},
		{
			name:         "uworm token (from Solana)",
			identifier:   "uworm",
			tokenChain:   1,
			tokenAddress: uworm,
			wrapped:      true,
		},
		{
			name:         "Too large token chain",
			identifier:   "wh/99999/165809739240a0ac03b98440fe8985548e3aa683cd0d4d9df5b5659669faa300",
			tokenChain:   0,
			tokenAddress: [32]byte{},
			wrapped:      false,
		},
		{
			name:         "Native token",
			identifier:   "asdasd",
			tokenChain:   0,
			tokenAddress: [32]byte{},
			wrapped:      false,
		},
		{
			name:         "Not matching format",
			identifier:   "wh/999/150794aa0b98440fe8985548e3aa683cd0d4d9df5b5659669faa300",
			tokenChain:   0,
			tokenAddress: [32]byte{},
			wrapped:      false,
		},
		{
			name:         "negative chain id",
			identifier:   "wh/-0222/165809739240a0ac03b98440fe8985548e3aa683cd0d4d9df5b5659669faa300",
			tokenChain:   0,
			tokenAddress: [32]byte{},
			wrapped:      false,
		},
		{
			name:         "base denom",
			identifier:   fmt.Sprintf("bwh/00008/%s", hex.EncodeToString(testToken[:])),
			tokenChain:   8,
			tokenAddress: testToken,
			wrapped:      true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenChain, tokenAddress, wrapped := GetWrappedCoinMeta(tt.identifier)
			require.EqualValues(t, tt.tokenChain, tokenChain)
			require.EqualValues(t, tt.tokenAddress, tokenAddress)
			require.EqualValues(t, tt.wrapped, wrapped)
		})
	}
}
