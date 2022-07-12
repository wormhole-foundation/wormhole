// This file contains the token and chain config to be used in the devnet environment.

package governor

import (
	"github.com/certusone/wormhole/node/pkg/vaa"
)

func (gov *ChainGovernor) initDevnetConfig() ([]tokenConfigEntry, []chainConfigEntry) {
	if gov.logger != nil {
		gov.logger.Info("cgov: setting up devnet config")
	}

	gov.dayLengthInMinutes = 5

	tokens := []tokenConfigEntry{
		tokenConfigEntry{chain: 1, addr: "069b8857feab8184fb687f634618c035dac439dc1aeb3b5598a0f00000000001", symbol: "SOL", coinGeckoId: "wrapped-solana", decimals: 8, price: 34.94}, // Addr: So11111111111111111111111111111111111111112, Notional: 4145006
		tokenConfigEntry{chain: 2, addr: "000000000000000000000000DDb64fE46a91D46ee29420539FC25FD07c5FEa3E", symbol: "WETH", coinGeckoId: "weth", decimals: 8, price: 1174},
	}

	chains := []chainConfigEntry{
		chainConfigEntry{emitterChainID: vaa.ChainIDSolana, emitterAddress: "c69a1b1a65dd336bf1df6a77afb501fc25db7fc0938cb08595a9ef473265cb4f", dailyLimit: 100},
		chainConfigEntry{emitterChainID: vaa.ChainIDEthereum, emitterAddress: "0x0290FB167208Af455bB137780163b7B7a9a10C16", dailyLimit: 100000},
	}

	return tokens, chains
}
