package keeper

import (
	"github.com/certusone/wormhole-chain/x/tokenbridge/types"
)

var _ types.QueryServer = Keeper{}
