// This file contains the token and chain config to be used in the mainnet environment.
//
// This file is maintained by hand. Add / remove / update entries as appropriate.

package governor

import (
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

func ChainList() []ChainConfigEntry {
	return []ChainConfigEntry{
		{EmitterChainID: vaa.ChainIDSolana, DailyLimit: 20_000_000, BigTransactionSize: 1_000_000},
		{EmitterChainID: vaa.ChainIDEthereum, DailyLimit: 10_000_000, BigTransactionSize: 1_000_000},
		{EmitterChainID: vaa.ChainIDBSC, DailyLimit: 1_000_000, BigTransactionSize: 100_000},
		{EmitterChainID: vaa.ChainIDPolygon, DailyLimit: 1_000_000, BigTransactionSize: 100_000},
		{EmitterChainID: vaa.ChainIDAvalanche, DailyLimit: 1_000_000, BigTransactionSize: 100_000},
		{EmitterChainID: vaa.ChainIDAlgorand, DailyLimit: 100_000, BigTransactionSize: 10_000},
		{EmitterChainID: vaa.ChainIDFantom, DailyLimit: 0, BigTransactionSize: 0},
		{EmitterChainID: vaa.ChainIDKlaytn, DailyLimit: 100_000, BigTransactionSize: 10_000},
		{EmitterChainID: vaa.ChainIDCelo, DailyLimit: 10_000, BigTransactionSize: 10_000},
		{EmitterChainID: vaa.ChainIDNear, DailyLimit: 100_000, BigTransactionSize: 10_000},
		{EmitterChainID: vaa.ChainIDMoonbeam, DailyLimit: 1_000_000, BigTransactionSize: 100_000},
		{EmitterChainID: vaa.ChainIDInjective, DailyLimit: 10_000, BigTransactionSize: 10_000},
		{EmitterChainID: vaa.ChainIDSui, DailyLimit: 2_500_000, BigTransactionSize: 250_000},
		{EmitterChainID: vaa.ChainIDAptos, DailyLimit: 1_000_000, BigTransactionSize: 100_000},
		{EmitterChainID: vaa.ChainIDArbitrum, DailyLimit: 2_500_000, BigTransactionSize: 250_000},
		{EmitterChainID: vaa.ChainIDOptimism, DailyLimit: 1_000_000, BigTransactionSize: 100_000},
		{EmitterChainID: vaa.ChainIDBase, DailyLimit: 2_500_000, BigTransactionSize: 250_000},
		{EmitterChainID: vaa.ChainIDSei, DailyLimit: 10_000, BigTransactionSize: 10_000},
		{EmitterChainID: vaa.ChainIDScroll, DailyLimit: 10_000, BigTransactionSize: 10_000},
		{EmitterChainID: vaa.ChainIDMantle, DailyLimit: 10_000, BigTransactionSize: 10_000},
		{EmitterChainID: vaa.ChainIDXLayer, DailyLimit: 0, BigTransactionSize: 0},
		{EmitterChainID: vaa.ChainIDBerachain, DailyLimit: 10_000, BigTransactionSize: 10_000},
		{EmitterChainID: vaa.ChainIDSeiEVM, DailyLimit: 500_000, BigTransactionSize: 50_000},
		{EmitterChainID: vaa.ChainIDWormchain, DailyLimit: 500_000, BigTransactionSize: 50_000},
		{EmitterChainID: vaa.ChainIDUnichain, DailyLimit: 250_000, BigTransactionSize: 25_000},
		{EmitterChainID: vaa.ChainIDWorldchain, DailyLimit: 10_000, BigTransactionSize: 10_000},
		{EmitterChainID: vaa.ChainIDInk, DailyLimit: 250_000, BigTransactionSize: 25_000},
		{EmitterChainID: vaa.ChainIDMezo, DailyLimit: 10_000, BigTransactionSize: 10_000},
		{EmitterChainID: vaa.ChainIDXRPLEVM, DailyLimit: 10_000, BigTransactionSize: 10_000},
		{EmitterChainID: vaa.ChainIDLinea, DailyLimit: 250_000, BigTransactionSize: 25_000},
		{EmitterChainID: vaa.ChainIDFogo, DailyLimit: 10_000, BigTransactionSize: 10_000},
		{EmitterChainID: vaa.ChainIDMonad, DailyLimit: 500_000, BigTransactionSize: 50_000},
		{EmitterChainID: vaa.ChainIDMegaETH, DailyLimit: 10_000, BigTransactionSize: 10_000},
		{EmitterChainID: vaa.ChainIDZeroGravity, DailyLimit: 10_000, BigTransactionSize: 10_000},
	}
}
