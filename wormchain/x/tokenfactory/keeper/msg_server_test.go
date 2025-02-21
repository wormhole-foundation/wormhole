package keeper_test

import (
	"fmt"
	"math/big"
	"time"

	"github.com/wormhole-foundation/wormchain/x/tokenfactory/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	denoms "github.com/wormhole-foundation/wormchain/x/tokenfactory/types"

	//banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	tmtypes "github.com/tendermint/tendermint/proto/tendermint/types"
	keeper "github.com/wormhole-foundation/wormchain/x/tokenfactory/keeper"
)

// TestMintDenomMsg tests TypeMsgMint message is emitted on a successful mint
func (suite *KeeperTestSuite) TestMintDenomMsg() {
	// Create a denom
	suite.CreateDefaultDenom()

	for _, tc := range []struct {
		desc                  string
		amount                int64
		mintDenom             string
		admin                 string
		valid                 bool
		expectedMessageEvents int
	}{
		{
			desc:      "denom does not exist",
			amount:    10,
			mintDenom: "factory/osmo1t7egva48prqmzl59x5ngv4zx0dtrwewc9m7z44/evmos",
			admin:     suite.TestAccs[0].String(),
			valid:     false,
		},
		{
			desc:                  "success case",
			amount:                10,
			mintDenom:             suite.defaultDenom,
			admin:                 suite.TestAccs[0].String(),
			valid:                 true,
			expectedMessageEvents: 1,
		},
	} {
		suite.Run(fmt.Sprintf("Case %s", tc.desc), func() {
			ctx := suite.Ctx.WithEventManager(sdk.NewEventManager())
			suite.Require().Equal(0, len(ctx.EventManager().Events()))
			// Test mint message
			suite.msgServer.Mint(sdk.WrapSDKContext(ctx), types.NewMsgMint(tc.admin, sdk.NewInt64Coin(tc.mintDenom, 10))) //nolint:errcheck
			// Ensure current number and type of event is emitted
			suite.AssertEventEmitted(ctx, types.TypeMsgMint, tc.expectedMessageEvents)
		})
	}
}

func (suite *KeeperTestSuite) TestMintHuge() {
	// Create a denom
	suite.CreateDefaultDenom()

	suite.Ctx = suite.App.BaseApp.NewContext(false, tmtypes.Header{Height: keeper.MainnetUseConditionalHeight, ChainID: "wormchain", Time: time.Now().UTC()})

	largeAmount := big.NewInt(0).Sub(big.NewInt(0).Exp(big.NewInt(2), big.NewInt(256), big.NewInt(0)), big.NewInt(1)) // (2 ** 256)-1
	belowLargeAmount := big.NewInt(0).Exp(big.NewInt(2), big.NewInt(191), big.NewInt(0))                              // 2 ** 191
	for _, tc := range []struct {
		desc                  string
		amount                sdk.Int
		mintDenom             string
		admin                 string
		valid                 bool
		expectedMessageEvents int
	}{
		{
			desc:                  "failure case - too many",
			amount:                sdk.NewIntFromBigInt(largeAmount),
			mintDenom:             suite.defaultDenom,
			admin:                 suite.TestAccs[0].String(),
			valid:                 false,
			expectedMessageEvents: 0,
		},
		{
			desc:                  "success case with 191",
			amount:                sdk.NewIntFromBigInt(belowLargeAmount),
			mintDenom:             suite.defaultDenom,
			admin:                 suite.TestAccs[0].String(),
			valid:                 true,
			expectedMessageEvents: 1,
		},
		{
			desc:                  "failure case - too many accumulated tokens",
			amount:                sdk.NewIntFromBigInt(belowLargeAmount),
			mintDenom:             suite.defaultDenom,
			admin:                 suite.TestAccs[0].String(),
			valid:                 false,
			expectedMessageEvents: 0,
		},
	} {
		suite.Run(fmt.Sprintf("Case %s", tc.desc), func() {
			ctx := suite.Ctx.WithEventManager(sdk.NewEventManager())
			suite.Require().Equal(0, len(ctx.EventManager().Events()))
			// Test mint message
			suite.msgServer.Mint(sdk.WrapSDKContext(ctx), types.NewMsgMint(tc.admin, sdk.NewCoin(tc.mintDenom, tc.amount))) //nolint:errcheck
			// Ensure current number and type of event is emitted
			suite.AssertEventEmitted(ctx, types.TypeMsgMint, tc.expectedMessageEvents)
		})
	}
}

func (suite *KeeperTestSuite) TestMintOffByOne() {
	// Create a denom
	suite.CreateDefaultDenom()
	suite.Ctx = suite.App.BaseApp.NewContext(false, tmtypes.Header{Height: keeper.MainnetUseConditionalHeight, ChainID: "wormchain", Time: time.Now().UTC()})

	for _, tc := range []struct {
		desc                  string
		amount                sdk.Int
		mintDenom             string
		admin                 string
		valid                 bool
		expectedMessageEvents int
	}{
		{
			desc:                  "failure case - too many plus 1",
			amount:                denoms.MintAmountLimit.Add(sdk.NewIntFromUint64(1)), // 2 ** 192 + 1
			mintDenom:             suite.defaultDenom,
			admin:                 suite.TestAccs[0].String(),
			valid:                 false,
			expectedMessageEvents: 0,
		},
		{
			desc:                  "failure case - too many exactly",
			amount:                denoms.MintAmountLimit, // 2 ** 192
			mintDenom:             suite.defaultDenom,
			admin:                 suite.TestAccs[0].String(),
			valid:                 true,
			expectedMessageEvents: 0,
		},
		{
			desc:                  "success case - one less than limit",
			amount:                denoms.MintAmountLimit.Sub(sdk.NewIntFromUint64(1)), // 2 ** 192 -1
			mintDenom:             suite.defaultDenom,
			admin:                 suite.TestAccs[0].String(),
			valid:                 true,
			expectedMessageEvents: 1,
		},
	} {
		suite.Run(fmt.Sprintf("Case %s", tc.desc), func() {
			ctx := suite.Ctx.WithEventManager(sdk.NewEventManager())
			suite.Require().Equal(0, len(ctx.EventManager().Events()))
			// Test mint message
			suite.msgServer.Mint(sdk.WrapSDKContext(ctx), types.NewMsgMint(tc.admin, sdk.NewCoin(tc.mintDenom, tc.amount))) //nolint:errcheck
			// Ensure current number and type of event is emitted
			suite.AssertEventEmitted(ctx, types.TypeMsgMint, tc.expectedMessageEvents)
		})
	}
}

func (suite *KeeperTestSuite) TestMintFixBlockHeightChecks() {
	// Create a denom
	suite.CreateDefaultDenom()

	test_cases := []struct {
		desc                  string
		amount                sdk.Int
		mintDenom             string
		admin                 string
		valid                 bool
		expectedMessageEvents int
	}{
		{
			desc:                  "success case - check not implemented before block height",
			amount:                denoms.MintAmountLimit, // 2 ** 192
			mintDenom:             suite.defaultDenom,
			admin:                 suite.TestAccs[0].String(),
			valid:                 true,
			expectedMessageEvents: 1,
		},
		{
			desc:                  "failure case - check implemented on specific block height",
			amount:                denoms.MintAmountLimit, // 2 ** 192
			mintDenom:             suite.defaultDenom,
			admin:                 suite.TestAccs[0].String(),
			valid:                 false,
			expectedMessageEvents: 0,
		},
		{
			desc:                  "failure case - check implemented after specific block height",
			amount:                denoms.MintAmountLimit, // 2 ** 192
			mintDenom:             suite.defaultDenom,
			admin:                 suite.TestAccs[0].String(),
			valid:                 false,
			expectedMessageEvents: 0,
		},
	}
	// Before the block has been reached. Should succeed with the call.
	suite.Ctx = suite.App.BaseApp.NewContext(false, tmtypes.Header{Height: keeper.MainnetUseConditionalHeight - 1, ChainID: "wormchain", Time: time.Now().UTC()})

	ctx := suite.Ctx.WithEventManager(sdk.NewEventManager())
	suite.Require().Equal(0, len(ctx.EventManager().Events()))
	// Test mint message
	suite.msgServer.Mint(sdk.WrapSDKContext(ctx), types.NewMsgMint(test_cases[0].admin, sdk.NewCoin(test_cases[0].mintDenom, test_cases[0].amount))) //nolint:errcheck

	// Ensure current number and type of event is emitted
	suite.AssertEventEmitted(ctx, types.TypeMsgMint, test_cases[0].expectedMessageEvents)

	// On the block has been reached
	suite.Ctx = suite.App.BaseApp.NewContext(false, tmtypes.Header{Height: keeper.MainnetUseConditionalHeight, ChainID: "wormchain", Time: time.Now().UTC()})
	ctx = suite.Ctx.WithEventManager(sdk.NewEventManager())
	suite.Require().Equal(0, len(ctx.EventManager().Events()))
	// Test mint message
	suite.msgServer.Mint(sdk.WrapSDKContext(ctx), types.NewMsgMint(test_cases[1].admin, sdk.NewCoin(test_cases[1].mintDenom, test_cases[1].amount))) //nolint:errcheck

	// Ensure current number and type of event is emitted
	suite.AssertEventEmitted(ctx, types.TypeMsgMint, test_cases[1].expectedMessageEvents)

	// After the block has been reached
	suite.Ctx = suite.App.BaseApp.NewContext(false, tmtypes.Header{Height: keeper.MainnetUseConditionalHeight + 1, ChainID: "wormchain", Time: time.Now().UTC()})
	ctx = suite.Ctx.WithEventManager(sdk.NewEventManager())
	suite.Require().Equal(0, len(ctx.EventManager().Events()))
	// Test mint message
	suite.msgServer.Mint(sdk.WrapSDKContext(ctx), types.NewMsgMint(test_cases[2].admin, sdk.NewCoin(test_cases[2].mintDenom, test_cases[2].amount))) //nolint:errcheck

	// Ensure current number and type of event is emitted
	suite.AssertEventEmitted(ctx, types.TypeMsgMint, test_cases[2].expectedMessageEvents)

}

// TestBurnDenomMsg tests TypeMsgBurn message is emitted on a successful burn
func (suite *KeeperTestSuite) TestBurnDenomMsg() {
	// Create a denom.
	suite.CreateDefaultDenom()
	// mint 10 default token for testAcc[0]
	suite.msgServer.Mint(sdk.WrapSDKContext(suite.Ctx), types.NewMsgMint(suite.TestAccs[0].String(), sdk.NewInt64Coin(suite.defaultDenom, 10))) //nolint:errcheck

	for _, tc := range []struct {
		desc                  string
		amount                int64
		burnDenom             string
		admin                 string
		valid                 bool
		expectedMessageEvents int
	}{
		{
			desc:      "denom does not exist",
			burnDenom: "factory/osmo1t7egva48prqmzl59x5ngv4zx0dtrwewc9m7z44/evmos",
			admin:     suite.TestAccs[0].String(),
			valid:     false,
		},
		{
			desc:                  "success case",
			burnDenom:             suite.defaultDenom,
			admin:                 suite.TestAccs[0].String(),
			valid:                 true,
			expectedMessageEvents: 1,
		},
	} {
		suite.Run(fmt.Sprintf("Case %s", tc.desc), func() {
			ctx := suite.Ctx.WithEventManager(sdk.NewEventManager())
			suite.Require().Equal(0, len(ctx.EventManager().Events()))
			// Test burn message
			suite.msgServer.Burn(sdk.WrapSDKContext(ctx), types.NewMsgBurn(tc.admin, sdk.NewInt64Coin(tc.burnDenom, 10))) //nolint:errcheck
			// Ensure current number and type of event is emitted
			suite.AssertEventEmitted(ctx, types.TypeMsgBurn, tc.expectedMessageEvents)
		})
	}
}

// TestCreateDenomMsg tests TypeMsgCreateDenom message is emitted on a successful denom creation
func (suite *KeeperTestSuite) TestCreateDenomMsg() {
	//defaultDenomCreationFee := types.Params{DenomCreationFee: sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(50000000)))}
	for _, tc := range []struct {
		desc string
		//denomCreationFee      types.Params
		subdenom              string
		valid                 bool
		expectedMessageEvents int
	}{
		{
			desc: "subdenom too long",
			//denomCreationFee: defaultDenomCreationFee,
			subdenom: "assadsadsadasdasdsadsadsadsadsadsadsklkadaskkkdasdasedskhanhassyeunganassfnlksdflksafjlkasd",
			valid:    false,
		},
		{
			desc: "success case: defaultDenomCreationFee",
			//denomCreationFee:      defaultDenomCreationFee,
			subdenom:              "evmos",
			valid:                 true,
			expectedMessageEvents: 1,
		},
	} {
		suite.SetupTest()
		suite.Run(fmt.Sprintf("Case %s", tc.desc), func() {
			//tokenFactoryKeeper := suite.App.TokenFactoryKeeper
			ctx := suite.Ctx.WithEventManager(sdk.NewEventManager())
			suite.Require().Equal(0, len(ctx.EventManager().Events()))
			// Set denom creation fee in params
			//tokenFactoryKeeper.SetParams(suite.Ctx, tc.denomCreationFee)
			// Test create denom message
			suite.msgServer.CreateDenom(sdk.WrapSDKContext(ctx), types.NewMsgCreateDenom(suite.TestAccs[0].String(), tc.subdenom)) //nolint:errcheck
			// Ensure current number and type of event is emitted
			suite.AssertEventEmitted(ctx, types.TypeMsgCreateDenom, tc.expectedMessageEvents)
		})
	}
}

// TestChangeAdminDenomMsg tests TypeMsgChangeAdmin message is emitted on a successful admin change
func (suite *KeeperTestSuite) TestChangeAdminDenomMsg() {
	for _, tc := range []struct {
		desc                    string
		msgChangeAdmin          func(denom string) *types.MsgChangeAdmin
		expectedChangeAdminPass bool
		expectedAdminIndex      int
		msgMint                 func(denom string) *types.MsgMint
		expectedMintPass        bool
		expectedMessageEvents   int
	}{
		{
			desc: "non-admins can't change the existing admin",
			msgChangeAdmin: func(denom string) *types.MsgChangeAdmin {
				return types.NewMsgChangeAdmin(suite.TestAccs[1].String(), denom, suite.TestAccs[2].String())
			},
			expectedChangeAdminPass: false,
			expectedAdminIndex:      0,
		},
		{
			desc: "success change admin",
			msgChangeAdmin: func(denom string) *types.MsgChangeAdmin {
				return types.NewMsgChangeAdmin(suite.TestAccs[0].String(), denom, suite.TestAccs[1].String())
			},
			expectedAdminIndex:      1,
			expectedChangeAdminPass: true,
			expectedMessageEvents:   1,
			msgMint: func(denom string) *types.MsgMint {
				return types.NewMsgMint(suite.TestAccs[1].String(), sdk.NewInt64Coin(denom, 5))
			},
			expectedMintPass: true,
		},
	} {
		suite.Run(fmt.Sprintf("Case %s", tc.desc), func() {
			// setup test
			suite.SetupTest()
			ctx := suite.Ctx.WithEventManager(sdk.NewEventManager())
			suite.Require().Equal(0, len(ctx.EventManager().Events()))
			// Create a denom and mint
			res, err := suite.msgServer.CreateDenom(sdk.WrapSDKContext(ctx), types.NewMsgCreateDenom(suite.TestAccs[0].String(), "bitcoin"))
			suite.Require().NoError(err)
			testDenom := res.GetNewTokenDenom()
			suite.msgServer.Mint(sdk.WrapSDKContext(ctx), types.NewMsgMint(suite.TestAccs[0].String(), sdk.NewInt64Coin(testDenom, 10))) //nolint:errcheck
			// Test change admin message
			suite.msgServer.ChangeAdmin(sdk.WrapSDKContext(ctx), tc.msgChangeAdmin(testDenom)) //nolint:errcheck
			// Ensure current number and type of event is emitted
			suite.AssertEventEmitted(ctx, types.TypeMsgChangeAdmin, tc.expectedMessageEvents)
		})
	}
}

// TestSetDenomMetaDataMsg tests TypeMsgSetDenomMetadata message is emitted on a successful denom metadata change
// Capability disabled
/*func (suite *KeeperTestSuite) TestSetDenomMetaDataMsg() {
	// setup test
	suite.SetupTest()
	suite.CreateDefaultDenom()

	for _, tc := range []struct {
		desc                  string
		msgSetDenomMetadata   types.MsgSetDenomMetadata
		expectedPass          bool
		expectedMessageEvents int
	}{
		{
			desc: "successful set denom metadata",
			msgSetDenomMetadata: *types.NewMsgSetDenomMetadata(suite.TestAccs[0].String(), banktypes.Metadata{
				Description: "yeehaw",
				DenomUnits: []*banktypes.DenomUnit{
					{
						Denom:    suite.defaultDenom,
						Exponent: 0,
					},
					{
						Denom:    "uosmo",
						Exponent: 6,
					},
				},
				Base:    suite.defaultDenom,
				Display: "uosmo",
				Name:    "OSMO",
				Symbol:  "OSMO",
			}),
			expectedPass:          false,
			expectedMessageEvents: 1,
		},
		{
			desc: "non existent factory denom name",
			msgSetDenomMetadata: *types.NewMsgSetDenomMetadata(suite.TestAccs[0].String(), banktypes.Metadata{
				Description: "yeehaw",
				DenomUnits: []*banktypes.DenomUnit{
					{
						Denom:    fmt.Sprintf("factory/%s/litecoin", suite.TestAccs[0].String()),
						Exponent: 0,
					},
					{
						Denom:    "uosmo",
						Exponent: 6,
					},
				},
				Base:    fmt.Sprintf("factory/%s/litecoin", suite.TestAccs[0].String()),
				Display: "uosmo",
				Name:    "OSMO",
				Symbol:  "OSMO",
			}),
			expectedPass: false,
		},
	} {
		suite.Run(fmt.Sprintf("Case %s", tc.desc), func() {
			ctx := suite.Ctx.WithEventManager(sdk.NewEventManager())
			suite.Require().Equal(0, len(ctx.EventManager().Events()))
			// Test set denom metadata message
			suite.msgServer.SetDenomMetadata(sdk.WrapSDKContext(ctx), &tc.msgSetDenomMetadata) //nolint:errcheck
			// Ensure current number and type of event is emitted
			suite.AssertEventEmitted(ctx, types.TypeMsgSetDenomMetadata, tc.expectedMessageEvents)
		})
	}
}*/
