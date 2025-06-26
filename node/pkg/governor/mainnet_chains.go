// This file contains the token and chain config to be used in the mainnet environment.
//
// This file is maintained by hand. Add / remove / update entries as appropriate.

package governor

import (
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

func ChainList() []ChainConfigEntry {
	return []ChainConfigEntry{
		{EmitterChainID: vaa.ChainIDSolana, DailyLimit: 50_000_000, BigTransactionSize: 2_500_000},
		{EmitterChainID: vaa.ChainIDEthereum, DailyLimit: 100_000_000, BigTransactionSize: 5_000_000},
		{EmitterChainID: vaa.ChainIDTerra, DailyLimit: 150_000, BigTransactionSize: 15_000},
		{EmitterChainID: vaa.ChainIDBSC, DailyLimit: 5_000_000, BigTransactionSize: 500_000},
		{EmitterChainID: vaa.ChainIDPolygon, DailyLimit: 5_000_000, BigTransactionSize: 500_000},
		{EmitterChainID: vaa.ChainIDAvalanche, DailyLimit: 5_000_000, BigTransactionSize: 500_000},
		{EmitterChainID: vaa.ChainIDOasis, DailyLimit: 250_000, BigTransactionSize: 25_000},
		{EmitterChainID: vaa.ChainIDAlgorand, DailyLimit: 750_000, BigTransactionSize: 75_000},
		{EmitterChainID: vaa.ChainIDAurora, DailyLimit: 0, BigTransactionSize: 0},
		{EmitterChainID: vaa.ChainIDFantom, DailyLimit: 500_000, BigTransactionSize: 50_000},
		{EmitterChainID: vaa.ChainIDKarura, DailyLimit: 150_000, BigTransactionSize: 15_000},
		{EmitterChainID: vaa.ChainIDAcala, DailyLimit: 100_000, BigTransactionSize: 10_000},
		{EmitterChainID: vaa.ChainIDKlaytn, DailyLimit: 500_000, BigTransactionSize: 50_000},
		{EmitterChainID: vaa.ChainIDCelo, DailyLimit: 1_000_000, BigTransactionSize: 100_000},
		{EmitterChainID: vaa.ChainIDNear, DailyLimit: 150_000, BigTransactionSize: 15_000},
		{EmitterChainID: vaa.ChainIDMoonbeam, DailyLimit: 5_000_000, BigTransactionSize: 500_000},
		{EmitterChainID: vaa.ChainIDTerra2, DailyLimit: 100_000, BigTransactionSize: 10_000},
		{EmitterChainID: vaa.ChainIDInjective, DailyLimit: 150_000, BigTransactionSize: 15_000},
		{EmitterChainID: vaa.ChainIDSui, DailyLimit: 10_000_000, BigTransactionSize: 500_000},
		{EmitterChainID: vaa.ChainIDAptos, DailyLimit: 1_000_000, BigTransactionSize: 100_000},
		{EmitterChainID: vaa.ChainIDArbitrum, DailyLimit: 5_000_000, BigTransactionSize: 500_000},
		{EmitterChainID: vaa.ChainIDOptimism, DailyLimit: 5_000_000, BigTransactionSize: 500_000},
		{EmitterChainID: vaa.ChainIDXpla, DailyLimit: 50_000, BigTransactionSize: 5_000},
		{EmitterChainID: vaa.ChainIDBase, DailyLimit: 5_000_000, BigTransactionSize: 500_000},
		{EmitterChainID: vaa.ChainIDSei, DailyLimit: 150_000, BigTransactionSize: 15_000},
		{EmitterChainID: vaa.ChainIDScroll, DailyLimit: 500_000, BigTransactionSize: 50_000},
		{EmitterChainID: vaa.ChainIDMantle, DailyLimit: 100_000, BigTransactionSize: 10_000},
		{EmitterChainID: vaa.ChainIDBlast, DailyLimit: 100_000, BigTransactionSize: 10_000},
		{EmitterChainID: vaa.ChainIDXLayer, DailyLimit: 100_000, BigTransactionSize: 10_000},
		{EmitterChainID: vaa.ChainIDBerachain, DailyLimit: 500_000, BigTransactionSize: 50_000},
		{EmitterChainID: vaa.ChainIDSeiEVM, DailyLimit: 500_000, BigTransactionSize: 50_000},
		{EmitterChainID: vaa.ChainIDWormchain, DailyLimit: 500_000, BigTransactionSize: 50_000},
		{EmitterChainID: vaa.ChainIDSnaxchain, DailyLimit: 500_000, BigTransactionSize: 50_000},
		{EmitterChainID: vaa.ChainIDUnichain, DailyLimit: 500_000, BigTransactionSize: 50_000},
		{EmitterChainID: vaa.ChainIDWorldchain, DailyLimit: 500_000, BigTransactionSize: 50_000},
		{EmitterChainID: vaa.ChainIDInk, DailyLimit: 500_000, BigTransactionSize: 50_000},
		{EmitterChainID: vaa.ChainIDMezo, DailyLimit: 500_000, BigTransactionSize: 50_000},
	}
}
