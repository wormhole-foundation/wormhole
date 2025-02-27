package evm

import (
	"fmt"
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

// TODO: Once this code is merged and verified to be stable, this test can be deleted.
func TestFinalityValuesForMainnet(t *testing.T) {
	testFinalityValuesForEnvironment(t, common.MainNet)
}

// TODO: Once this code is merged and verified to be stable, this test can be deleted.
func TestFinalityValuesForTestnet(t *testing.T) {
	testFinalityValuesForEnvironment(t, common.TestNet)
}

// TODO: Once this code is merged and verified to be stable, this function can be deleted.
func testFinalityValuesForEnvironment(t *testing.T, env common.Environment) {
	t.Helper()
	m, err := GetChainConfigMap(env)
	require.NoError(t, err)

	for chainID, entry := range m {
		t.Run(chainID.String(), func(t *testing.T) {
			finalized, safe, err := getFinalityForTest(env, chainID)
			require.NoError(t, err)
			assert.Equal(t, finalized, entry.Finalized)
			assert.Equal(t, safe, entry.Safe)
		})
	}
}

// getFinalityForTest was lifted from the old `getFinality` watcher function so we could validate our config data.
// TODO: Once this code is merged and verified to be stable, this function can be deleted so we don't have to maintain it.
func getFinalityForTest(env common.Environment, chainID vaa.ChainID) (finalized bool, safe bool, err error) {
	// Tilt supports polling for both finalized and safe.
	if env == common.UnsafeDevNet {
		finalized = true
		safe = true

		// The following chains support polling for both finalized and safe.
	} else if chainID == vaa.ChainIDAcala ||
		chainID == vaa.ChainIDArbitrum ||
		chainID == vaa.ChainIDArbitrumSepolia ||
		chainID == vaa.ChainIDBase ||
		chainID == vaa.ChainIDBaseSepolia ||
		chainID == vaa.ChainIDBlast ||
		chainID == vaa.ChainIDBSC ||
		chainID == vaa.ChainIDEthereum ||
		chainID == vaa.ChainIDHolesky ||
		chainID == vaa.ChainIDHyperEVM ||
		chainID == vaa.ChainIDInk ||
		chainID == vaa.ChainIDKarura ||
		chainID == vaa.ChainIDMantle ||
		chainID == vaa.ChainIDMonad ||
		chainID == vaa.ChainIDMoonbeam ||
		chainID == vaa.ChainIDOptimism ||
		chainID == vaa.ChainIDOptimismSepolia ||
		chainID == vaa.ChainIDSeiEVM ||
		chainID == vaa.ChainIDSepolia ||
		chainID == vaa.ChainIDSnaxchain ||
		chainID == vaa.ChainIDUnichain ||
		chainID == vaa.ChainIDWorldchain ||
		chainID == vaa.ChainIDXLayer {
		finalized = true
		safe = true

	} else if chainID == vaa.ChainIDCelo {
		// TODO: Celo testnet now supports finalized and safe. As of January 2025, mainnet doesn't yet support safe. Once Celo mainnet cuts over, Celo can
		// be added to the list above. That change won't be super urgent since we'll just continue to publish safe as finalized, which is not a huge deal.
		finalized = true
		safe = env != common.MainNet

		// Polygon now supports polling for finalized but not safe.
		// https://forum.polygon.technology/t/optimizing-decentralized-apps-ux-with-milestones-a-significantly-accelerated-finality-solution/13154
	} else if chainID == vaa.ChainIDPolygon ||
		chainID == vaa.ChainIDPolygonSepolia {
		finalized = true

		// As of 11/10/2023 Scroll supports polling for finalized but not safe.
	} else if chainID == vaa.ChainIDScroll {
		finalized = true

		// As of 9/06/2024 Linea supports polling for finalized but not safe.
	} else if chainID == vaa.ChainIDLinea {
		finalized = true

		// The following chains support instant finality.
	} else if chainID == vaa.ChainIDAvalanche ||
		chainID == vaa.ChainIDBerachain || // Berachain supports instant finality: https://docs.berachain.com/faq/
		chainID == vaa.ChainIDOasis ||
		chainID == vaa.ChainIDAurora ||
		chainID == vaa.ChainIDFantom ||
		chainID == vaa.ChainIDKlaytn {
		return false, false, nil

		// Anything else is undefined / not supported.
	} else {
		return false, false, fmt.Errorf("unsupported chain: %s", chainID.String())
	}

	return
}
