package vaa

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChainID_String(t *testing.T) {
	tests := []struct {
		name string
		c    ChainID
		want string
	}{
		{name: "unset", c: ChainIDUnset, want: "unset"},
		{name: "solana", c: ChainIDSolana, want: "solana"},
		{name: "ethereum", c: ChainIDEthereum, want: "ethereum"},
		{name: "bsc", c: ChainIDBSC, want: "bsc"},
		{name: "polygon", c: ChainIDPolygon, want: "polygon"},
		{name: "avalanche", c: ChainIDAvalanche, want: "avalanche"},
		{name: "unknown", c: ChainID(9999), want: "unknown chain ID: 9999"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.c.String())
		})
	}
}

func TestChainIDFromString(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    ChainID
		wantErr bool
	}{
		{name: "solana", input: "solana", want: ChainIDSolana, wantErr: false},
		{name: "ethereum", input: "ethereum", want: ChainIDEthereum, wantErr: false},
		{name: "bsc", input: "bsc", want: ChainIDBSC, wantErr: false},
		{name: "sui", input: "sui", want: ChainIDSui, wantErr: false},
		{name: "wormchain", input: "wormchain", want: ChainIDWormchain, wantErr: false},
		{name: "base", input: "base", want: ChainIDBase, wantErr: false},
		{name: "case insensitive", input: "SoLaNa", want: ChainIDSolana, wantErr: false},
		{name: "unknown chain", input: "nonexistentchain", want: ChainIDUnset, wantErr: true},
		{name: "empty string", input: "", want: ChainIDUnset, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ChainIDFromString(tt.input)
			assert.Equal(t, tt.want, got)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestChainIDFromString_RoundTrip(t *testing.T) {
	// Verify that String -> ChainIDFromString is consistent for all known chains
	for _, id := range GetAllNetworkIDs() {
		t.Run(id.String(), func(t *testing.T) {
			roundTrip, err := ChainIDFromString(id.String())
			require.NoError(t, err)
			assert.Equal(t, id, roundTrip)
		})
	}
}
