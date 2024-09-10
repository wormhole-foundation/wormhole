package sdk

import (
	"errors"
	"strings"

	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

var ErrInvalidEnv = errors.New("invalid environment")
var ErrNotFound = errors.New("not found")

// IsEvmChainID if the specified chain is defined as an EVM chain ID in the specified environment.
func IsEvmChainID(env string, chainID vaa.ChainID) (bool, error) {
	var m *map[vaa.ChainID]int
	if env == "prod" || env == "mainnet" {
		m = &MainnetEvmChainIDs
	} else if env == "test" || env == "testnet" {
		m = &TestnetEvmChainIDs
	} else {
		return false, ErrInvalidEnv
	}
	_, exists := (*m)[chainID]
	return exists, nil
}

// GetEvmChainID returns the expected EVM chain ID associated with the given Wormhole chain ID and environment passed it.
func GetEvmChainID(env string, chainID vaa.ChainID) (int, error) {
	env = strings.ToLower(env)
	if env == "prod" || env == "mainnet" {
		return getEvmChainID(MainnetEvmChainIDs, chainID)
	}
	if env == "test" || env == "testnet" {
		return getEvmChainID(TestnetEvmChainIDs, chainID)
	}
	return 0, ErrInvalidEnv
}

func getEvmChainID(evmChains map[vaa.ChainID]int, chainID vaa.ChainID) (int, error) {
	id, exists := evmChains[chainID]
	if !exists {
		return 0, ErrNotFound
	}
	return id, nil
}
