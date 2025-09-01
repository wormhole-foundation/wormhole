// This file contains the token and chain config to be used in the testnet environment.

package governor

import (
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

func (gov *ChainGovernor) initTestnetConfig() ([]TokenConfigEntry, []TokenConfigEntry, []ChainConfigEntry, []corridor) {
	gov.logger.Info("setting up testnet config")

	tokens := []TokenConfigEntry{
		{Chain: 1, Addr: "069b8857feab8184fb687f634618c035dac439dc1aeb3b5598a0f00000000001", Symbol: "SOL", CoinGeckoId: "wrapped-solana", Decimals: 9, Price: 138.11}, // Addr: So11111111111111111111111111111111111111112, Notional: 82226686.73036034
		{Chain: 1, Addr: "3b442cb3912157f13a933d0134282d032b5ffecd01a2dbf1b7790608df002ea7", Symbol: "USDC", CoinGeckoId: "usd-coin", Decimals: 6, Price: 1.001},       // Addr: 4zMMC9srt5Ri5X14GAgXhaHii3GnPAEERYPJgZJDncDU, Notional: 6780118.197035182
	}

	flowCancelTokens := []TokenConfigEntry{}
	flowCancelCorridors := []corridor{}

	if gov.flowCancelEnabled {
		flowCancelTokens = []TokenConfigEntry{
			{Chain: 1, Addr: "3b442cb3912157f13a933d0134282d032b5ffecd01a2dbf1b7790608df002ea7", Symbol: "USDC", CoinGeckoId: "usd-coin", Decimals: 6, Price: 1.001}, // Addr: 4zMMC9srt5Ri5X14GAgXhaHii3GnPAEERYPJgZJDncDU, Notional: 6780118.197035182
		}
		flowCancelCorridors = []corridor{
			{first: vaa.ChainIDEthereum, second: vaa.ChainIDSui},
		}
	}

	chains := []ChainConfigEntry{
		{EmitterChainID: vaa.ChainIDSolana, DailyLimit: 100000000},
		{EmitterChainID: vaa.ChainIDEthereum, DailyLimit: 100000000},
		{EmitterChainID: vaa.ChainIDFantom, DailyLimit: 1000000},
	}

	return tokens, flowCancelTokens, chains, flowCancelCorridors
}
