// This file contains the token and chain config to be used in the testnet environment.

package governor

import (
	"github.com/certusone/wormhole/node/pkg/vaa"
)

func (gov *ChainGovernor) initTestnetConfig() ([]tokenConfigEntry, []chainConfigEntry) {
	if gov.logger != nil {
		gov.logger.Info("cgov: setting up testnet config")
	}

	tokens := []tokenConfigEntry{
		tokenConfigEntry{chain: 2, addr: "000000000000000000000000b4fbf271143f4fbf7b91a5ded31805e42b2208d6", symbol: "WETH", coinGeckoId: "weth", decimals: 8, price: 1174},
	}

	chains := []chainConfigEntry{
		chainConfigEntry{emitterChainID: vaa.ChainIDSolana, dailyLimit: 1000},
		chainConfigEntry{emitterChainID: vaa.ChainIDEthereum, dailyLimit: 100000},
	}

	return tokens, chains
}
