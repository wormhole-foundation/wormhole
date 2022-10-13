// This file contains the token and chain config to be used in the mainnet environment.
//
// This file is maintained by hand. Add / remove / update entries as appropriate.

package governor

import (
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

func chainList() []chainConfigEntry {
	return []chainConfigEntry{
		chainConfigEntry{emitterChainID: vaa.ChainIDSolana, dailyLimit: 50_000_000, bigTransactionSize: 5_000_000},
		chainConfigEntry{emitterChainID: vaa.ChainIDEthereum, dailyLimit: 50_000_000, bigTransactionSize: 5_000_000},
		chainConfigEntry{emitterChainID: vaa.ChainIDTerra, dailyLimit: 5_000_000, bigTransactionSize: 500_000},
		chainConfigEntry{emitterChainID: vaa.ChainIDBSC, dailyLimit: 5_000_000, bigTransactionSize: 500_000},
		chainConfigEntry{emitterChainID: vaa.ChainIDPolygon, dailyLimit: 5_000_000, bigTransactionSize: 500_000},
		chainConfigEntry{emitterChainID: vaa.ChainIDAvalanche, dailyLimit: 5_000_000, bigTransactionSize: 500_000},
		chainConfigEntry{emitterChainID: vaa.ChainIDOasis, dailyLimit: 2_000_000, bigTransactionSize: 500_000},
		chainConfigEntry{emitterChainID: vaa.ChainIDAlgorand, dailyLimit: 200_000, bigTransactionSize: 20_000},
		chainConfigEntry{emitterChainID: vaa.ChainIDAurora, dailyLimit: 500_000, bigTransactionSize: 50_000},
		chainConfigEntry{emitterChainID: vaa.ChainIDFantom, dailyLimit: 500_000, bigTransactionSize: 50_000},
		chainConfigEntry{emitterChainID: vaa.ChainIDKarura, dailyLimit: 500_000, bigTransactionSize: 50_000},
		chainConfigEntry{emitterChainID: vaa.ChainIDAcala, dailyLimit: 500_000, bigTransactionSize: 50_000},
		chainConfigEntry{emitterChainID: vaa.ChainIDKlaytn, dailyLimit: 5_000_000, bigTransactionSize: 500_000},
		chainConfigEntry{emitterChainID: vaa.ChainIDCelo, dailyLimit: 5_000_000, bigTransactionSize: 500_000},
		chainConfigEntry{emitterChainID: vaa.ChainIDNear, dailyLimit: 5_000_000, bigTransactionSize: 500_000},
		chainConfigEntry{emitterChainID: vaa.ChainIDTerra2, dailyLimit: 500_000, bigTransactionSize: 50_000},
		chainConfigEntry{emitterChainID: vaa.ChainIDMoonbeam, dailyLimit: 200_000, bigTransactionSize: 20_000},
		chainConfigEntry{emitterChainID: vaa.ChainIDXpla, dailyLimit: 200_000, bigTransactionSize: 20_000},
	}
}
