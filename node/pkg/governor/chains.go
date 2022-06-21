// This file is maintained by hand. Add / remove / update entries as appropriate.
package governor

import (
	"github.com/certusone/wormhole/node/pkg/vaa"
)

func chainList() []chainConfigEntry {
	return []chainConfigEntry{
		chainConfigEntry{emitterChainID: vaa.ChainIDEthereum, dailyLimit: 1000000},
		chainConfigEntry{emitterChainID: vaa.ChainIDPolygon, dailyLimit: 1000000},
	}
}
