// This file contains the token and chain config to be used in the testnet environment.

package governor

import (
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

func (gov *ChainGovernor) initTestnetConfig() ([]tokenConfigEntry, []tokenConfigEntry, []chainConfigEntry, []pipe) {
	gov.logger.Info("setting up testnet config")

	tokens := []tokenConfigEntry{
		{chain: 1, addr: "069b8857feab8184fb687f634618c035dac439dc1aeb3b5598a0f00000000001", symbol: "SOL", coinGeckoId: "wrapped-solana", decimals: 9, price: 138.11}, // Addr: So11111111111111111111111111111111111111112, Notional: 82226686.73036034
		{chain: 1, addr: "3b442cb3912157f13a933d0134282d032b5ffecd01a2dbf1b7790608df002ea7", symbol: "USDC", coinGeckoId: "usd-coin", decimals: 6, price: 1.001},       // Addr: 4zMMC9srt5Ri5X14GAgXhaHii3GnPAEERYPJgZJDncDU, Notional: 6780118.197035182
	}

	flowCancelTokens := []tokenConfigEntry{}
	flowCancelPipes := []pipe{}

	if gov.flowCancelEnabled {
		flowCancelTokens = []tokenConfigEntry{
			{chain: 1, addr: "3b442cb3912157f13a933d0134282d032b5ffecd01a2dbf1b7790608df002ea7", symbol: "USDC", coinGeckoId: "usd-coin", decimals: 6, price: 1.001}, // Addr: 4zMMC9srt5Ri5X14GAgXhaHii3GnPAEERYPJgZJDncDU, Notional: 6780118.197035182
		}
		flowCancelPipes = []pipe{
			{first: vaa.ChainIDEthereum, second: vaa.ChainIDSui},
		}
	}

	chains := []chainConfigEntry{
		{emitterChainID: vaa.ChainIDSolana, dailyLimit: 100000000},
		{emitterChainID: vaa.ChainIDEthereum, dailyLimit: 100000000},
		{emitterChainID: vaa.ChainIDFantom, dailyLimit: 1000000},
	}

	return tokens, flowCancelTokens, chains, flowCancelPipes
}
