package keeper

import (
	"github.com/certusone/wormhole-chain/x/wormhole/types"
)

var _ types.QueryServer = Keeper{}
