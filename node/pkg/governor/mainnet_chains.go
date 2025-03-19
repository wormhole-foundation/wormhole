// This file contains the token and chain config to be used in the mainnet environment.
//
// This file is maintained by hand. Add / remove / update entries as appropriate.

package governor

import (
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

func chainList() []chainConfigEntry {
	return []chainConfigEntry{
		{emitterChainID: vaa.ChainIDSolana, dailyLimit: 50_000_000, bigTransactionSize: 2_500_000},
		{emitterChainID: vaa.ChainIDEthereum, dailyLimit: 100_000_000, bigTransactionSize: 5_000_000},
		{emitterChainID: vaa.ChainIDTerra, dailyLimit: 150_000, bigTransactionSize: 15_000},
		{emitterChainID: vaa.ChainIDBSC, dailyLimit: 5_000_000, bigTransactionSize: 500_000},
		{emitterChainID: vaa.ChainIDPolygon, dailyLimit: 5_000_000, bigTransactionSize: 500_000},
		{emitterChainID: vaa.ChainIDAvalanche, dailyLimit: 5_000_000, bigTransactionSize: 500_000},
		{emitterChainID: vaa.ChainIDOasis, dailyLimit: 250_000, bigTransactionSize: 25_000},
		{emitterChainID: vaa.ChainIDAlgorand, dailyLimit: 1_000_000, bigTransactionSize: 100_000},
		{emitterChainID: vaa.ChainIDAurora, dailyLimit: 0, bigTransactionSize: 0},
		{emitterChainID: vaa.ChainIDFantom, dailyLimit: 500_000, bigTransactionSize: 50_000},
		{emitterChainID: vaa.ChainIDKarura, dailyLimit: 150_000, bigTransactionSize: 15_000},
		{emitterChainID: vaa.ChainIDAcala, dailyLimit: 100_000, bigTransactionSize: 10_000},
		{emitterChainID: vaa.ChainIDKlaytn, dailyLimit: 500_000, bigTransactionSize: 50_000},
		{emitterChainID: vaa.ChainIDCelo, dailyLimit: 2_000_000, bigTransactionSize: 200_000},
		{emitterChainID: vaa.ChainIDNear, dailyLimit: 150_000, bigTransactionSize: 15_000},
		{emitterChainID: vaa.ChainIDMoonbeam, dailyLimit: 5_000_000, bigTransactionSize: 500_000},
		{emitterChainID: vaa.ChainIDTerra2, dailyLimit: 100_000, bigTransactionSize: 10_000},
		{emitterChainID: vaa.ChainIDInjective, dailyLimit: 500_000, bigTransactionSize: 50_000},
		{emitterChainID: vaa.ChainIDSui, dailyLimit: 10_000_000, bigTransactionSize: 500_000},
		{emitterChainID: vaa.ChainIDAptos, dailyLimit: 1_000_000, bigTransactionSize: 100_000},
		{emitterChainID: vaa.ChainIDArbitrum, dailyLimit: 5_000_000, bigTransactionSize: 500_000},
		{emitterChainID: vaa.ChainIDOptimism, dailyLimit: 5_000_000, bigTransactionSize: 500_000},
		{emitterChainID: vaa.ChainIDXpla, dailyLimit: 50_000, bigTransactionSize: 5_000},
		{emitterChainID: vaa.ChainIDBase, dailyLimit: 5_000_000, bigTransactionSize: 500_000},
		{emitterChainID: vaa.ChainIDSei, dailyLimit: 5_000_000, bigTransactionSize: 500_000},
		{emitterChainID: vaa.ChainIDScroll, dailyLimit: 500_000, bigTransactionSize: 50_000},
		{emitterChainID: vaa.ChainIDMantle, dailyLimit: 100_000, bigTransactionSize: 10_000},
		{emitterChainID: vaa.ChainIDBlast, dailyLimit: 100_000, bigTransactionSize: 10_000},
		{emitterChainID: vaa.ChainIDXLayer, dailyLimit: 100_000, bigTransactionSize: 10_000},
		{emitterChainID: vaa.ChainIDBerachain, dailyLimit: 500_000, bigTransactionSize: 50_000},
		{emitterChainID: vaa.ChainIDSeiEVM, dailyLimit: 500_000, bigTransactionSize: 50_000},
		{emitterChainID: vaa.ChainIDWormchain, dailyLimit: 500_000, bigTransactionSize: 50_000},
		{emitterChainID: vaa.ChainIDSnaxchain, dailyLimit: 500_000, bigTransactionSize: 50_000},
		{emitterChainID: vaa.ChainIDUnichain, dailyLimit: 500_000, bigTransactionSize: 50_000},
		{emitterChainID: vaa.ChainIDWorldchain, dailyLimit: 500_000, bigTransactionSize: 50_000},
		{emitterChainID: vaa.ChainIDInk, dailyLimit: 500_000, bigTransactionSize: 50_000},
	}
}
