package common

import (
	"testing"

	"github.com/certusone/wormhole/node/pkg/readiness"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"

	"github.com/stretchr/testify/assert"
)

const (
	// Ethereum is defined in readiness.go.
	ReadinessSolanaSyncing    readiness.Component = "solanaSyncing"
	ReadinessTerraSyncing     readiness.Component = "terraSyncing"
	ReadinessAlgorandSyncing  readiness.Component = "algorandSyncing"
	ReadinessNearSyncing      readiness.Component = "nearSyncing"
	ReadinessAptosSyncing     readiness.Component = "aptosSyncing"
	ReadinessSuiSyncing       readiness.Component = "suiSyncing"
	ReadinessBSCSyncing       readiness.Component = "bscSyncing"
	ReadinessPolygonSyncing   readiness.Component = "polygonSyncing"
	ReadinessAvalancheSyncing readiness.Component = "avalancheSyncing"
	ReadinessOasisSyncing     readiness.Component = "oasisSyncing"
	ReadinessAuroraSyncing    readiness.Component = "auroraSyncing"
	ReadinessFantomSyncing    readiness.Component = "fantomSyncing"
	ReadinessKaruraSyncing    readiness.Component = "karuraSyncing"
	ReadinessAcalaSyncing     readiness.Component = "acalaSyncing"
	ReadinessKlaytnSyncing    readiness.Component = "klaytnSyncing"
	ReadinessCeloSyncing      readiness.Component = "celoSyncing"
	ReadinessMoonbeamSyncing  readiness.Component = "moonbeamSyncing"
	ReadinessNeonSyncing      readiness.Component = "neonSyncing"
	ReadinessTerra2Syncing    readiness.Component = "terra2Syncing"
	ReadinessInjectiveSyncing readiness.Component = "injectiveSyncing"
	ReadinessXplaSyncing      readiness.Component = "xplaSyncing"
	ReadinessPythNetSyncing   readiness.Component = "pythnetSyncing"
	ReadinessArbitrumSyncing  readiness.Component = "arbitrumSyncing"
	ReadinessOptimismSyncing  readiness.Component = "optimismSyncing"
	ReadinessBaseSyncing      readiness.Component = "baseSyncing"
	ReadinessWormchainSyncing readiness.Component = "wormchainSyncing"
)

// This test is just to make sure that nothing got broken when we switched from manually specifying the readiness syncing labels.
// Once this functionality is merged, this test can probably be deleted (so that we don't need to keep adding new chains going forward).
func TestConvertChainIdToReadinessSyncing(t *testing.T) {
	type test struct {
		input  vaa.ChainID
		output readiness.Component
	}

	// Positive Test Cases
	p_tests := []test{
		{input: vaa.ChainIDSolana, output: ReadinessSolanaSyncing},
		{input: vaa.ChainIDEthereum, output: ReadinessEthSyncing},
		{input: vaa.ChainIDTerra, output: ReadinessTerraSyncing},
		{input: vaa.ChainIDBSC, output: ReadinessBSCSyncing},
		{input: vaa.ChainIDPolygon, output: ReadinessPolygonSyncing},
		{input: vaa.ChainIDAvalanche, output: ReadinessAvalancheSyncing},
		{input: vaa.ChainIDOasis, output: ReadinessOasisSyncing},
		{input: vaa.ChainIDAlgorand, output: ReadinessAlgorandSyncing},
		{input: vaa.ChainIDAptos, output: ReadinessAptosSyncing},
		{input: vaa.ChainIDSui, output: ReadinessSuiSyncing},
		{input: vaa.ChainIDNear, output: ReadinessNearSyncing},
		{input: vaa.ChainIDAurora, output: ReadinessAuroraSyncing},
		{input: vaa.ChainIDFantom, output: ReadinessFantomSyncing},
		{input: vaa.ChainIDKarura, output: ReadinessKaruraSyncing},
		{input: vaa.ChainIDAcala, output: ReadinessAcalaSyncing},
		{input: vaa.ChainIDKlaytn, output: ReadinessKlaytnSyncing},
		{input: vaa.ChainIDCelo, output: ReadinessCeloSyncing},
		{input: vaa.ChainIDMoonbeam, output: ReadinessMoonbeamSyncing},
		{input: vaa.ChainIDNeon, output: ReadinessNeonSyncing},
		{input: vaa.ChainIDTerra2, output: ReadinessTerra2Syncing},
		{input: vaa.ChainIDInjective, output: ReadinessInjectiveSyncing},
		{input: vaa.ChainIDArbitrum, output: ReadinessArbitrumSyncing},
		{input: vaa.ChainIDPythNet, output: ReadinessPythNetSyncing},
		{input: vaa.ChainIDOptimism, output: ReadinessOptimismSyncing},
		{input: vaa.ChainIDXpla, output: ReadinessXplaSyncing},
		// BTC readiness not defined yet {input: vaa.ChainIDBtc, output: ReadinessBtcSyncing},
		{input: vaa.ChainIDBase, output: ReadinessBaseSyncing},
	}

	// Negative Test Cases
	n_tests := []test{
		{input: vaa.ChainIDUnset, output: ""},
	}

	for _, tc := range p_tests {
		t.Run(tc.input.String(), func(t *testing.T) {
			chainId, err := ConvertChainIdToReadinessSyncing(tc.input)
			assert.Equal(t, tc.output, chainId)
			assert.NoError(t, err)
		})
	}

	for _, tc := range n_tests {
		t.Run(tc.input.String(), func(t *testing.T) {
			chainId, err := ConvertChainIdToReadinessSyncing(tc.input)
			assert.Equal(t, tc.output, chainId)
			assert.Error(t, err)
		})
	}
}

func TestMustRegisterReadinessSyncing(t *testing.T) {
	// The first time should work.
	assert.NotPanics(t, func() {
		MustRegisterReadinessSyncing(vaa.ChainIDEthereum)
	})

	// A second time should panic.
	assert.Panics(t, func() {
		MustRegisterReadinessSyncing(vaa.ChainIDEthereum)
	})

	// An invalid chainID should panic.
	assert.Panics(t, func() {
		MustRegisterReadinessSyncing(vaa.ChainIDUnset)
	})
}
