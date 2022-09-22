// This file contains the token and chain config to be used in the mainnet environment.
//
// This file is maintained by hand. Add / remove / update entries as appropriate.

package governor

import (
	"github.com/certusone/wormhole/node/pkg/vaa"
)

func chainList() []chainConfigEntry {
	return []chainConfigEntry{
		{emitterChainID: vaa.ChainIDSolana, dailyLimit: 50_000_000, bigTransactionSize: 5_000_000},
		{emitterChainID: vaa.ChainIDEthereum, dailyLimit: 50_000_000, bigTransactionSize: 5_000_000},
		{emitterChainID: vaa.ChainIDTerra, dailyLimit: 1_000_000, bigTransactionSize: 100_000},
		{emitterChainID: vaa.ChainIDBSC, dailyLimit: 5_000_000, bigTransactionSize: 500_000},
		{emitterChainID: vaa.ChainIDPolygon, dailyLimit: 5_000_000, bigTransactionSize: 500_000},
		{emitterChainID: vaa.ChainIDAvalanche, dailyLimit: 5_000_000, bigTransactionSize: 500_000},
		{emitterChainID: vaa.ChainIDOasis, dailyLimit: 500_000, bigTransactionSize: 50_000},
		{emitterChainID: vaa.ChainIDAlgorand, dailyLimit: 200_000, bigTransactionSize: 20_000},
		{emitterChainID: vaa.ChainIDAurora, dailyLimit: 500_000, bigTransactionSize: 50_000},
		{emitterChainID: vaa.ChainIDFantom, dailyLimit: 500_000, bigTransactionSize: 50_000},
		{emitterChainID: vaa.ChainIDKarura, dailyLimit: 500_000, bigTransactionSize: 50_000},
		{emitterChainID: vaa.ChainIDAcala, dailyLimit: 500_000, bigTransactionSize: 50_000},
		{emitterChainID: vaa.ChainIDKlaytn, dailyLimit: 1_000_000, bigTransactionSize: 100_000},
		{emitterChainID: vaa.ChainIDCelo, dailyLimit: 1_000_000, bigTransactionSize: 100_000},
		{emitterChainID: vaa.ChainIDNear, dailyLimit: 200_000, bigTransactionSize: 20_000},
		{emitterChainID: vaa.ChainIDTerra2, dailyLimit: 500_000, bigTransactionSize: 50_000},
		{emitterChainID: vaa.ChainIDAptos, dailyLimit: 50_000, bigTransactionSize: 5_000},
	}
}
