// This file contains the token and chain config to be used in the mainnet environment.
//
// This file is maintained by hand. Add / remove / update entries as appropriate.

package governor

import (
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

func ChainList() []ChainConfigEntry {
	return []ChainConfigEntry{
		{EmitterChainID: vaa.ChainIDSolana, USDLimit: 50_000_000, BigTransactionSize: 10_000_000},
		{EmitterChainID: vaa.ChainIDEthereum, USDLimit: 100_000_000, BigTransactionSize: 20_000_000},
		{EmitterChainID: vaa.ChainIDBSC, USDLimit: 5_000_000, BigTransactionSize: 500_000},
		{EmitterChainID: vaa.ChainIDPolygon, USDLimit: 5_000_000, BigTransactionSize: 500_000},
		{EmitterChainID: vaa.ChainIDAvalanche, USDLimit: 5_000_000, BigTransactionSize: 500_000},
		{EmitterChainID: vaa.ChainIDAlgorand, USDLimit: 750_000, BigTransactionSize: 75_000},
		{EmitterChainID: vaa.ChainIDFantom, USDLimit: 500_000, BigTransactionSize: 50_000},
		{EmitterChainID: vaa.ChainIDKlaytn, USDLimit: 500_000, BigTransactionSize: 50_000},
		{EmitterChainID: vaa.ChainIDCelo, USDLimit: 1_000_000, BigTransactionSize: 100_000},
		{EmitterChainID: vaa.ChainIDNear, USDLimit: 150_000, BigTransactionSize: 15_000},
		{EmitterChainID: vaa.ChainIDMoonbeam, USDLimit: 5_000_000, BigTransactionSize: 500_000},
		{EmitterChainID: vaa.ChainIDInjective, USDLimit: 150_000, BigTransactionSize: 15_000},
		{EmitterChainID: vaa.ChainIDSui, USDLimit: 10_000_000, BigTransactionSize: 2_000_000},
		{EmitterChainID: vaa.ChainIDAptos, USDLimit: 1_000_000, BigTransactionSize: 100_000},
		{EmitterChainID: vaa.ChainIDArbitrum, USDLimit: 5_000_000, BigTransactionSize: 2_000_000},
		{EmitterChainID: vaa.ChainIDOptimism, USDLimit: 5_000_000, BigTransactionSize: 500_000},
		{EmitterChainID: vaa.ChainIDBase, USDLimit: 5_000_000, BigTransactionSize: 2_000_000},
		{EmitterChainID: vaa.ChainIDSei, USDLimit: 150_000, BigTransactionSize: 15_000},
		{EmitterChainID: vaa.ChainIDScroll, USDLimit: 500_000, BigTransactionSize: 50_000},
		{EmitterChainID: vaa.ChainIDMantle, USDLimit: 100_000, BigTransactionSize: 10_000},
		{EmitterChainID: vaa.ChainIDXLayer, USDLimit: 0, BigTransactionSize: 0},
		{EmitterChainID: vaa.ChainIDBerachain, USDLimit: 500_000, BigTransactionSize: 50_000},
		{EmitterChainID: vaa.ChainIDSeiEVM, USDLimit: 500_000, BigTransactionSize: 50_000},
		{EmitterChainID: vaa.ChainIDWormchain, USDLimit: 500_000, BigTransactionSize: 50_000},
		{EmitterChainID: vaa.ChainIDUnichain, USDLimit: 500_000, BigTransactionSize: 50_000},
		{EmitterChainID: vaa.ChainIDWorldchain, USDLimit: 500_000, BigTransactionSize: 50_000},
		{EmitterChainID: vaa.ChainIDInk, USDLimit: 500_000, BigTransactionSize: 50_000},
		{EmitterChainID: vaa.ChainIDMezo, USDLimit: 500_000, BigTransactionSize: 50_000},
		{EmitterChainID: vaa.ChainIDXRPLEVM, USDLimit: 500_000, BigTransactionSize: 50_000},
		{EmitterChainID: vaa.ChainIDLinea, USDLimit: 500_000, BigTransactionSize: 50_000},
		{EmitterChainID: vaa.ChainIDFogo, USDLimit: 500_000, BigTransactionSize: 50_000},
		{EmitterChainID: vaa.ChainIDMonad, USDLimit: 5_000_000, BigTransactionSize: 500_000},
	}
}
