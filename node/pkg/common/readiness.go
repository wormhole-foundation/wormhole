package common

import (
	"fmt"

	"github.com/certusone/wormhole/node/pkg/readiness"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

const (
	ReadinessEthSyncing       readiness.Component = "ethSyncing"
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

// ChainIdToReadinessSyncing maps a chain ID to a readiness syncing value. It will panic if the chain ID is invalid
// so it should only be used during initialization. Otherwise use ChainIdToReadinessSyncingWithError.
func ChainIdToReadinessSyncing(chainID vaa.ChainID) readiness.Component {
	ret, err := ChainIdToReadinessSyncingWithError(chainID)
	if err != nil {
		panic(err)
	}

	return ret
}

// ChainIdToReadinessSyncingWithError maps a chain ID to a readiness syncing value. It returns an error if the chain ID is invalid.
func ChainIdToReadinessSyncingWithError(chainID vaa.ChainID) (readiness.Component, error) {
	if chainID == vaa.ChainIDEthereum {
		// The readiness for Ethereum is "ethSyncing", not "ethereumSyncing". Don't know if changing it will break monitoring. . .
		return ReadinessEthSyncing, nil
	}
	if _, err := vaa.ChainIDFromString(chainID.String()); err != nil {
		return readiness.Component(""), fmt.Errorf("invalid chainID: %d", uint16(chainID))
	}
	return readiness.Component(chainID.String() + "Syncing"), nil
}
