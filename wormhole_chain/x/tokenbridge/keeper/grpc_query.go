package keeper

import (
	"github.com/wormhole-foundation/wormhole-chain/x/tokenbridge/types"
)

var _ types.QueryServer = Keeper{}
