package keeper_test

import (
	"context"
	"testing"

	keepertest "github.com/certusone/wormhole-chain/testutil/keeper"
	"github.com/certusone/wormhole-chain/x/tokenbridge/keeper"
	"github.com/certusone/wormhole-chain/x/tokenbridge/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func setupMsgServer(t testing.TB) (types.MsgServer, context.Context) {
	k, ctx := keepertest.TokenbridgeKeeper(t)
	return keeper.NewMsgServerImpl(*k), sdk.WrapSDKContext(ctx)
}
