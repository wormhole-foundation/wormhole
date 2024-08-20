package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/wormhole-foundation/wormchain/app/apptesting"
	"github.com/wormhole-foundation/wormchain/x/tokenfactory/keeper"
	"github.com/wormhole-foundation/wormchain/x/tokenfactory/types"
)

type KeeperTestSuite struct {
	apptesting.KeeperTestHelper

	queryClient     types.QueryClient
	bankQueryClient banktypes.QueryClient
	msgServer       types.MsgServer
	// defaultDenom is on the suite, as it depends on the creator test address.
	defaultDenom string
}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(KeeperTestSuite))
}

func (suite *KeeperTestSuite) SetupTest() {
	suite.Setup()

	// DenomCreationFee is nil, only the secondary denom is funded
	// OUTDATED: Fund every TestAcc with two denoms, one of which is the denom creation fee
	// OUTDATED: fundAccsAmount := sdk.NewCoins(sdk.NewCoin(types.DefaultParams().DenomCreationFee[0].Denom, types.DefaultParams().DenomCreationFee[0].Amount.MulRaw(100)), sdk.NewCoin(apptesting.SecondaryDenom, apptesting.SecondaryAmount))

	fundAccsAmount := sdk.NewCoins(sdk.NewCoin(apptesting.SecondaryDenom, apptesting.SecondaryAmount))
	for _, acc := range suite.TestAccs {
		suite.FundAcc(acc, fundAccsAmount)
	}

	suite.queryClient = types.NewQueryClient(suite.QueryHelper)
	suite.bankQueryClient = banktypes.NewQueryClient(suite.QueryHelper)
	suite.msgServer = keeper.NewMsgServerImpl(suite.App.TokenFactoryKeeper)
}

func (suite *KeeperTestSuite) CreateDefaultDenom() {
	res, _ := suite.msgServer.CreateDenom(sdk.WrapSDKContext(suite.Ctx), types.NewMsgCreateDenom(suite.TestAccs[0].String(), "bitcoin"))
	suite.defaultDenom = res.GetNewTokenDenom()
}

func (suite *KeeperTestSuite) TestCreateModuleAccount() {
	wormchain := suite.App

	// remove module account
	tokenfactoryModuleAccount := wormchain.AccountKeeper.GetAccount(suite.Ctx, wormchain.AccountKeeper.GetModuleAddress(types.ModuleName))
	wormchain.AccountKeeper.RemoveAccount(suite.Ctx, tokenfactoryModuleAccount)

	// ensure module account was removed
	suite.Ctx = wormchain.BaseApp.NewContext(false, tmproto.Header{ChainID: "testing"})
	tokenfactoryModuleAccount = wormchain.AccountKeeper.GetAccount(suite.Ctx, wormchain.AccountKeeper.GetModuleAddress(types.ModuleName))
	suite.Require().Nil(tokenfactoryModuleAccount)

	// create module account
	wormchain.TokenFactoryKeeper.CreateModuleAccount(suite.Ctx)

	// check that the module account is now initialized
	tokenfactoryModuleAccount = wormchain.AccountKeeper.GetAccount(suite.Ctx, wormchain.AccountKeeper.GetModuleAddress(types.ModuleName))
	suite.Require().NotNil(tokenfactoryModuleAccount)
}
