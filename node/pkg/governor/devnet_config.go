// This file contains the token and chain config to be used in the devnet environment.

package governor

import (
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

func (gov *ChainGovernor) initDevnetConfig() ([]tokenConfigEntry, []tokenConfigEntry, []chainConfigEntry) {
	gov.logger.Info("setting up devnet config")

	gov.dayLengthInMinutes = 5

	tokens := []tokenConfigEntry{
		{chain: 1, addr: "069b8857feab8184fb687f634618c035dac439dc1aeb3b5598a0f00000000001", symbol: "SOL", coinGeckoId: "wrapped-solana", decimals: 9, price: 138.11}, // Addr: So11111111111111111111111111111111111111112, Notional: 82226686.73036034
		{chain: 1, addr: "c6fa7af3bedbad3a3d65f36aabc97431b1bbe4c2d2f6e0e47ca60203452f5d61", symbol: "USDC", coinGeckoId: "usd-coin", decimals: 6, price: 1.001},       // Addr: EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v, Notional: 6780118.197035182
		{chain: 2, addr: "000000000000000000000000DDb64fE46a91D46ee29420539FC25FD07c5FEa3E", symbol: "WETH", coinGeckoId: "weth", decimals: 8, price: 1174},
	}

	flowCancelTokens := []tokenConfigEntry{}
	if gov.flowCancelEnabled {
		flowCancelTokens = []tokenConfigEntry{
			{chain: 1, addr: "c6fa7af3bedbad3a3d65f36aabc97431b1bbe4c2d2f6e0e47ca60203452f5d61", symbol: "USDC", coinGeckoId: "usd-coin", decimals: 6, price: 1.001}, // Addr: EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v, Notional: 6780118.197035182
		}
	}

	chains := []chainConfigEntry{
		{emitterChainID: vaa.ChainIDSolana, dailyLimit: 100, bigTransactionSize: 75},
		{emitterChainID: vaa.ChainIDEthereum, dailyLimit: 100000},
	}

	return tokens, flowCancelTokens, chains
}
