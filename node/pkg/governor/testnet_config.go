// This file contains the token and chain config to be used in the testnet environment.

package governor

import (
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

func (gov *ChainGovernor) initTestnetConfig() ([]tokenConfigEntry, []chainConfigEntry) {
	gov.logger.Info("setting up testnet config")

	tokens := []tokenConfigEntry{
		tokenConfigEntry{chain: 1, addr: "069b8857feab8184fb687f634618c035dac439dc1aeb3b5598a0f00000000001", symbol: "SOL", coinGeckoId: "wrapped-solana", decimals: 8, price: 34.94}, // Addr: So11111111111111111111111111111111111111112, Notional: 4145006
	}

	chains := []chainConfigEntry{
		chainConfigEntry{emitterChainID: vaa.ChainIDSolana, dailyLimit: 100000000},
		chainConfigEntry{emitterChainID: vaa.ChainIDEthereum, dailyLimit: 100000000},
		chainConfigEntry{emitterChainID: vaa.ChainIDFantom, dailyLimit: 1000000},
	}

	return tokens, chains
}
