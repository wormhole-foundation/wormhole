package common

import (
	"fmt"

	"github.com/certusone/wormhole/node/pkg/readiness"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

const (
	ReadinessEthSyncing readiness.Component = "ethSyncing"
)

// MustRegisterReadinessSyncing registers the specified chain for readiness syncing and returns the readiness syncing value.
// This function will panic if the chain ID is invalid so it should only be used during initialization.
func MustRegisterReadinessSyncing(chainID vaa.ChainID) readiness.Component {
	readinessSync, err := ChainIdToReadinessSyncing(chainID)
	if err != nil {
		panic(err)
	}

	// This will panic if the component is already registered.
	readiness.RegisterComponent(readinessSync)
	return readinessSync
}

// ChainIdToReadinessSyncing maps a chain ID to a readiness syncing value. It returns an error if the chain ID is invalid.
func ChainIdToReadinessSyncing(chainID vaa.ChainID) (readiness.Component, error) {
	if chainID == vaa.ChainIDEthereum {
		// The readiness for Ethereum is "ethSyncing", not "ethereumSyncing". Changing it would most likely break monitoring. . .
		return ReadinessEthSyncing, nil
	}
	str := chainID.String()
	if _, err := vaa.ChainIDFromString(str); err != nil {
		return readiness.Component(""), fmt.Errorf("invalid chainID: %d", uint16(chainID))
	}
	return readiness.Component(str + "Syncing"), nil
}
