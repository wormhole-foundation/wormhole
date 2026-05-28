package cosmwasm

import (
	"testing"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/stretchr/testify/assert"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

// TestNewWatcher_B64EncodedByChain locks in the per-chain b64Encoded selection
// in NewWatcher (watcher.go:125). Sei mainnet must be base64-encoded; flipping
// it to false would silently break event parsing in production, since the
// watcher's reobservation path uses this flag to decode event keys/values.
func TestNewWatcher_B64EncodedByChain(t *testing.T) {
	cases := []struct {
		name       string
		chainID    vaa.ChainID
		env        common.Environment
		wantB64    bool
		wantLogKey string
	}{
		{"Sei mainnet", vaa.ChainIDSei, common.MainNet, true, "_contract_address"},
		{"Sei testnet", vaa.ChainIDSei, common.TestNet, true, "_contract_address"},
		{"Sei devnet always base64", vaa.ChainIDSei, common.UnsafeDevNet, true, "_contract_address"},
		{"Injective mainnet not base64", vaa.ChainIDInjective, common.MainNet, false, "_contract_address"},
		{"Injective devnet base64", vaa.ChainIDInjective, common.UnsafeDevNet, true, "_contract_address"},
		{"Terra2 mainnet not base64", vaa.ChainIDTerra2, common.MainNet, false, "_contract_address"},
		{"Terra2 devnet base64", vaa.ChainIDTerra2, common.UnsafeDevNet, true, "_contract_address"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := NewWatcher("ws://unused", "http://unused", "contract", nil, nil, tc.chainID, tc.env)
			assert.Equal(t, tc.wantB64, w.b64Encoded, "b64Encoded mismatch")
			assert.Equal(t, tc.wantLogKey, w.contractAddressLogKey, "contractAddressLogKey mismatch")
			assert.Equal(t, tc.chainID, w.chainID)
		})
	}
}
