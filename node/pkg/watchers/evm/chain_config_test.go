package evm

import (
	"testing"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

func TestSupportedInMainnet(t *testing.T) {
	assert.True(t, SupportedInMainnet(vaa.ChainIDEthereum))
	assert.False(t, SupportedInMainnet(vaa.ChainIDSepolia))
}

func TestGetEvmChainID(t *testing.T) {
	type test struct {
		env    common.Environment
		input  vaa.ChainID
		output uint64
		err    error
	}

	// Note: Don't intend to list every chain here, just enough to verify `GetEvmChainID`.
	tests := []test{
		{env: common.MainNet, input: vaa.ChainIDUnset, err: ErrNotFound},
		{env: common.MainNet, input: vaa.ChainIDSepolia, err: ErrNotFound},
		{env: common.MainNet, input: vaa.ChainIDEthereum, output: 1},
		{env: common.MainNet, input: vaa.ChainIDBSC, output: 56},
		{env: common.TestNet, input: vaa.ChainIDUnset, err: ErrNotFound},
		{env: common.TestNet, input: vaa.ChainIDSepolia, output: 11155111},
		{env: common.TestNet, input: vaa.ChainIDEthereum, output: 17000},
		{env: common.GoTest, input: vaa.ChainIDEthereum, err: ErrInvalidEnv},
	}

	for _, tc := range tests {
		t.Run(string(tc.env)+"-"+tc.input.String(), func(t *testing.T) {
			evmChainID, err := GetEvmChainID(tc.env, tc.input)
			if tc.err != nil {
				assert.ErrorIs(t, tc.err, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.output, evmChainID)
			}
		})
	}
}

func TestGetFinality(t *testing.T) {
	type test struct {
		env       common.Environment
		input     vaa.ChainID
		finalized bool
		safe      bool
		err       error
	}

	// Note: Don't intend to list every chain here, just enough to verify `GetEvmChainID`.
	tests := []test{
		{env: common.MainNet, input: vaa.ChainIDUnset, err: ErrNotFound},
		{env: common.MainNet, input: vaa.ChainIDSepolia, err: ErrNotFound},
		{env: common.MainNet, input: vaa.ChainIDEthereum, finalized: true, safe: true},
		{env: common.MainNet, input: vaa.ChainIDBSC, finalized: true, safe: true},
		{env: common.MainNet, input: vaa.ChainIDScroll, finalized: true, safe: false},
		{env: common.TestNet, input: vaa.ChainIDUnset, err: ErrNotFound},
		{env: common.TestNet, input: vaa.ChainIDSepolia, finalized: true, safe: true},
		{env: common.TestNet, input: vaa.ChainIDEthereum, finalized: true, safe: true},
		{env: common.GoTest, input: vaa.ChainIDEthereum, err: ErrInvalidEnv},
	}

	for _, tc := range tests {
		t.Run(string(tc.env)+"-"+tc.input.String(), func(t *testing.T) {
			finalized, safe, err := GetFinality(tc.env, tc.input)
			if tc.err != nil {
				assert.ErrorIs(t, tc.err, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.finalized, finalized)
				assert.Equal(t, tc.safe, safe)
			}
		})
	}
}
