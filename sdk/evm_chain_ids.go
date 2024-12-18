package sdk

import (
	"errors"
	"strings"

	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

type EvmChainIDs map[vaa.ChainID]int

var mainnetEvmChainIDs = EvmChainIDs{
	vaa.ChainIDAcala:     787,
	vaa.ChainIDArbitrum:  42161,
	vaa.ChainIDAurora:    1313161554,
	vaa.ChainIDAvalanche: 43114,
	vaa.ChainIDBSC:       56,
	vaa.ChainIDBase:      8453,
	vaa.ChainIDBlast:     81457,
	vaa.ChainIDCelo:      42220,
	vaa.ChainIDEthereum:  1,
	vaa.ChainIDFantom:    250,
	vaa.ChainIDGnosis:    100,
	vaa.ChainIDKarura:    686,
	vaa.ChainIDKlaytn:    8217,
	vaa.ChainIDLinea:     0, // TODO: We need this value
	vaa.ChainIDMantle:    5000,
	vaa.ChainIDMoonbeam:  1284,
	vaa.ChainIDOasis:     42262,
	vaa.ChainIDOptimism:  10,
	vaa.ChainIDPolygon:   137,
	vaa.ChainIDRootstock: 30,
	vaa.ChainIDScroll:    534352,
	vaa.ChainIDSnaxchain: 2192,
	vaa.ChainIDXLayer:    196,
}

var testnetEvmChainIDs = EvmChainIDs{
	vaa.ChainIDAcala:           597,
	vaa.ChainIDArbitrum:        421613,
	vaa.ChainIDArbitrumSepolia: 421614,
	vaa.ChainIDAurora:          1313161555,
	vaa.ChainIDAvalanche:       43113,
	vaa.ChainIDBSC:             97,
	vaa.ChainIDBase:            84531,
	vaa.ChainIDBaseSepolia:     84532,
	vaa.ChainIDBerachain:       80084,
	vaa.ChainIDBlast:           168587773,
	vaa.ChainIDCelo:            44787,
	vaa.ChainIDEthereum:        17000, // This is actually the value for Holesky, since Goerli obsolete.
	vaa.ChainIDFantom:          4002,
	vaa.ChainIDGnosis:          77,
	vaa.ChainIDHolesky:         17000,
	vaa.ChainIDKarura:          596,
	vaa.ChainIDKlaytn:          1001,
	vaa.ChainIDLinea:           59141,
	vaa.ChainIDMantle:          5003,
	vaa.ChainIDMoonbeam:        1287,
	vaa.ChainIDOasis:           42261,
	vaa.ChainIDOptimism:        420,
	vaa.ChainIDOptimismSepolia: 11155420,
	vaa.ChainIDPolygon:         80001,
	vaa.ChainIDPolygonSepolia:  80002,
	vaa.ChainIDRootstock:       31,
	vaa.ChainIDScroll:          534353,
	vaa.ChainIDSeiEVM:          713715,
	vaa.ChainIDSepolia:         11155111,
	vaa.ChainIDSnaxchain:       13001,
	vaa.ChainIDXLayer:          195,
}

var ErrInvalidEnv = errors.New("invalid environment")
var ErrNotFound = errors.New("not found")

// GetEvmChainID returns the expected EVM chain ID associated with the given Wormhole chain ID and environment passed it.
func GetEvmChainID(env string, chainID vaa.ChainID) (int, error) {
	env = strings.ToLower(env)
	if env == "prod" || env == "mainnet" {
		return getEvmChainID(mainnetEvmChainIDs, chainID)
	}
	if env == "test" || env == "testnet" {
		return getEvmChainID(testnetEvmChainIDs, chainID)
	}
	return 0, ErrInvalidEnv
}

func getEvmChainID(evmChains EvmChainIDs, chainID vaa.ChainID) (int, error) {
	id, exists := evmChains[chainID]
	if !exists {
		return 0, ErrNotFound
	}
	return id, nil
}
