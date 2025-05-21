package evm

import (
	"encoding/hex"
	"strings"
	"testing"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"

	ethCommon "github.com/ethereum/go-ethereum/common"
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

func TestMainnetContractAddresses(t *testing.T) {
	verifyContractAddresses(t, mainnetChainConfig)
	verifyContractAddresses(t, testnetChainConfig)
}

func verifyContractAddresses(t *testing.T, m EnvMap) {
	t.Helper()
	zeroAddr := ethCommon.HexToAddress("0x0")
	for chainId, entry := range m {
		t.Run(chainId.String(), func(t *testing.T) {
			// It must be set.
			require.NotEqual(t, "", entry.ContractAddr)

			// Since `ethCommon.HexToAddress` never fails, make sure a regular hex conversion works.
			_, err := hex.DecodeString(strings.TrimPrefix(entry.ContractAddr, "0x"))
			require.NoError(t, err)

			// Don't allow it to be empty / the zero address.
			require.NotEqual(t, zeroAddr, ethCommon.HexToAddress(entry.ContractAddr))
		})
	}
}

func TestGetContractAddr(t *testing.T) {
	type test struct {
		env    common.Environment
		input  vaa.ChainID
		output string
		err    error
	}

	// Note: Don't intend to list every chain here, just enough to verify `GetContractAddrString`.
	tests := []test{
		{env: common.MainNet, input: vaa.ChainIDUnset, err: ErrNotFound},
		{env: common.MainNet, input: vaa.ChainIDSepolia, err: ErrNotFound},
		{env: common.MainNet, input: vaa.ChainIDEthereum, output: "0x98f3c9e6E3fAce36bAAd05FE09d375Ef1464288B"},
		{env: common.MainNet, input: vaa.ChainIDBSC, output: "0x98f3c9e6E3fAce36bAAd05FE09d375Ef1464288B"},
		{env: common.TestNet, input: vaa.ChainIDUnset, err: ErrNotFound},
		{env: common.TestNet, input: vaa.ChainIDSepolia, output: "0x4a8bc80Ed5a4067f1CCf107057b8270E0cC11A78"},
		{env: common.TestNet, input: vaa.ChainIDEthereum, output: "0xa10f2eF61dE1f19f586ab8B6F2EbA89bACE63F7a"},
		{env: common.GoTest, input: vaa.ChainIDEthereum, err: ErrInvalidEnv},
	}

	for _, tc := range tests {
		t.Run(string(tc.env)+"-"+tc.input.String(), func(t *testing.T) {
			str, err := GetContractAddrString(tc.env, tc.input)
			if tc.err != nil {
				require.ErrorIs(t, tc.err, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.output, str)
			}

			addr, err := GetContractAddr(tc.env, tc.input)
			if tc.err != nil {
				assert.ErrorIs(t, tc.err, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, ethCommon.HexToAddress(tc.output), addr)
			}
		})
	}
}
