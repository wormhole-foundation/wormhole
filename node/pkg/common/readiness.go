package common

import (
	"fmt"

	"github.com/certusone/wormhole/node/pkg/readiness"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

const (
	ReadinessEthSyncing readiness.Component = "ethSyncing"
	ReadinessIBCSyncing readiness.Component = "IBCSyncing"
)

// MustRegisterReadinessSyncing registers the specified chain for readiness syncing. It panics if the chain ID is invalid so it should only be used during initialization.
// TODO: Using vaa.ChainID is bad here because there can be multiple watchers for the same chainId, e.g. solana-finalized and solana-confirmed. This is currently handled as a special case for solana in node/node.go, but should really be fixed here.
func MustRegisterReadinessSyncing(chainID vaa.ChainID) {
	readiness.RegisterComponent(MustConvertChainIdToReadinessSyncing(chainID))
}

// MustConvertChainIdToReadinessSyncing maps a chain ID to a readiness syncing value. It panics if the chain ID is invalid so it should only be used during initialization.
func MustConvertChainIdToReadinessSyncing(chainID vaa.ChainID) readiness.Component {
	readinessSync, err := ConvertChainIdToReadinessSyncing(chainID)
	if err != nil {
		panic(err)
	}
	return readinessSync
}

// ConvertChainIdToReadinessSyncing maps a chain ID to a readiness syncing value. It returns an error if the chain ID is invalid.
func ConvertChainIdToReadinessSyncing(chainID vaa.ChainID) (readiness.Component, error) {
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
