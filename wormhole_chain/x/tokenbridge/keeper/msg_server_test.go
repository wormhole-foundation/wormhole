package keeper_test

import (
	"context"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	keepertest "github.com/wormhole-foundation/wormhole-chain/testutil/keeper"
	"github.com/wormhole-foundation/wormhole-chain/x/tokenbridge/keeper"
	"github.com/wormhole-foundation/wormhole-chain/x/tokenbridge/types"
)

func setupMsgServer(t testing.TB) (types.MsgServer, context.Context) {
	k, ctx := keepertest.TokenbridgeKeeper(t)
	return keeper.NewMsgServerImpl(*k), sdk.WrapSDKContext(ctx)
}
