package keeper

import (
	"github.com/wormhole-foundation/wormchain/x/wormhole/types"
)

var _ types.QueryServer = Keeper{}
