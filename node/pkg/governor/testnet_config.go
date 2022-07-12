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
		tokenConfigEntry{chain: 1, addr: "069b8857feab8184fb687f634618c035dac439dc1aeb3b5598a0f00000000001", symbol: "SOL", coinGeckoId: "wrapped-solana", decimals: 8, price: 34.94}, // Addr: So11111111111111111111111111111111111111112, Notional: 4145006
	}

	chains := []chainConfigEntry{
		chainConfigEntry{emitterChainID: vaa.ChainIDFantom, emitterAddress: "0x599CEa2204B4FaECd584Ab1F2b6aCA137a0afbE8", dailyLimit: 100},
		chainConfigEntry{emitterChainID: vaa.ChainIDOasis, emitterAddress: "0x88d8004A9BdbfD9D28090A02010C19897a29605c", dailyLimit: 1000000},
	}

	return tokens, chains
}
