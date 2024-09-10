package sdk

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

func TestGetEvmChainID(t *testing.T) {
	type test struct {
		env    string
		input  vaa.ChainID
		output int
		err    error
	}

	// Note: Don't intend to list every chain here, just enough to verify `GetEvmChainID`.
	tests := []test{
		{env: "mainnet", input: vaa.ChainIDUnset, output: 0, err: ErrNotFound},
		{env: "mainnet", input: vaa.ChainIDSepolia, output: 0, err: ErrNotFound},
		{env: "mainnet", input: vaa.ChainIDEthereum, output: 1},
		{env: "mainnet", input: vaa.ChainIDArbitrum, output: 42161},
		{env: "testnet", input: vaa.ChainIDSepolia, output: 11155111},
		{env: "testnet", input: vaa.ChainIDEthereum, output: 17000},
		{env: "junk", input: vaa.ChainIDEthereum, output: 17000, err: ErrInvalidEnv},
	}

	for _, tc := range tests {
		t.Run(tc.env+"-"+tc.input.String(), func(t *testing.T) {
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
func TestIsEvmChainID(t *testing.T) {
	type test struct {
		env    string
		input  vaa.ChainID
		output bool
		err    error
	}

	// Note: Don't intend to list every chain here, just enough to verify `GetEvmChainID`.
	tests := []test{
		{env: "mainnet", input: vaa.ChainIDUnset, output: false},
		{env: "mainnet", input: vaa.ChainIDSepolia, output: false},
		{env: "mainnet", input: vaa.ChainIDEthereum, output: true},
		{env: "mainnet", input: vaa.ChainIDArbitrum, output: true},
		{env: "mainnet", input: vaa.ChainIDSolana, output: false},
		{env: "testnet", input: vaa.ChainIDSepolia, output: true},
		{env: "testnet", input: vaa.ChainIDEthereum, output: true},
		{env: "testnet", input: vaa.ChainIDTerra, output: false},
		{env: "junk", input: vaa.ChainIDEthereum, output: true, err: ErrInvalidEnv},
	}

	for _, tc := range tests {
		t.Run(tc.env+"-"+tc.input.String(), func(t *testing.T) {
			result, err := IsEvmChainID(tc.env, tc.input)
			if tc.err != nil {
				assert.ErrorIs(t, tc.err, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.output, result)
			}
		})
	}
}
