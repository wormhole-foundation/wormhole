package keeper

import (
	"github.com/wormhole-foundation/wormhole-chain/x/wormhole/types"
)

var _ types.QueryServer = Keeper{}
