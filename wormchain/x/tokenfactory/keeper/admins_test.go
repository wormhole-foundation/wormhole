package keeper_test

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	//banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/wormhole-foundation/wormchain/x/tokenfactory/types"
)

func (suite *KeeperTestSuite) TestAdminMsgs() {
	addr0bal := int64(0)
	addr1bal := int64(0)

	bankKeeper := suite.App.BankKeeper

	suite.CreateDefaultDenom()
	// Make sure that the admin is set correctly
	queryRes, err := suite.queryClient.DenomAuthorityMetadata(suite.Ctx.Context(), &types.QueryDenomAuthorityMetadataRequest{
		Denom: suite.defaultDenom,
	})
	suite.Require().NoError(err)
	suite.Require().Equal(suite.TestAccs[0].String(), queryRes.AuthorityMetadata.Admin)

	// Test minting to admins own account
	_, err = suite.msgServer.Mint(sdk.WrapSDKContext(suite.Ctx), types.NewMsgMint(suite.TestAccs[0].String(), sdk.NewInt64Coin(suite.defaultDenom, 10)))
	addr0bal += 10
	suite.Require().NoError(err)
	suite.Require().True(bankKeeper.GetBalance(suite.Ctx, suite.TestAccs[0], suite.defaultDenom).Amount.Int64() == addr0bal, bankKeeper.GetBalance(suite.Ctx, suite.TestAccs[0], suite.defaultDenom))

	// Test minting to a different account
	_, err = suite.msgServer.Mint(sdk.WrapSDKContext(suite.Ctx), types.NewMsgMintTo(suite.TestAccs[0].String(), sdk.NewInt64Coin(suite.defaultDenom, 10), suite.TestAccs[1].String()))
	addr1bal += 10
	suite.Require().NoError(err)
	suite.Require().True(suite.App.BankKeeper.GetBalance(suite.Ctx, suite.TestAccs[1], suite.defaultDenom).Amount.Int64() == addr1bal, suite.App.BankKeeper.GetBalance(suite.Ctx, suite.TestAccs[1], suite.defaultDenom))

	// Test force transferring
	// Capability is disabled
	/*_, err = suite.msgServer.ForceTransfer(sdk.WrapSDKContext(suite.Ctx), types.NewMsgForceTransfer(suite.TestAccs[0].String(), sdk.NewInt64Coin(suite.defaultDenom, 5), suite.TestAccs[1].String(), suite.TestAccs[0].String()))
	addr1bal -= 5
	addr0bal += 5
	suite.Require().NoError(err)
	suite.Require().True(suite.App.BankKeeper.GetBalance(suite.Ctx, suite.TestAccs[0], suite.defaultDenom).Amount.Int64() == addr0bal, suite.App.BankKeeper.GetBalance(suite.Ctx, suite.TestAccs[0], suite.defaultDenom))
	suite.Require().True(suite.App.BankKeeper.GetBalance(suite.Ctx, suite.TestAccs[1], suite.defaultDenom).Amount.Int64() == addr1bal, suite.App.BankKeeper.GetBalance(suite.Ctx, suite.TestAccs[1], suite.defaultDenom))*/

	// Test burning from own account
	_, err = suite.msgServer.Burn(sdk.WrapSDKContext(suite.Ctx), types.NewMsgBurn(suite.TestAccs[0].String(), sdk.NewInt64Coin(suite.defaultDenom, 5)))
	suite.Require().NoError(err)
	suite.Require().True(bankKeeper.GetBalance(suite.Ctx, suite.TestAccs[1], suite.defaultDenom).Amount.Int64() == addr1bal)

	// Test Change Admin
	_, err = suite.msgServer.ChangeAdmin(sdk.WrapSDKContext(suite.Ctx), types.NewMsgChangeAdmin(suite.TestAccs[0].String(), suite.defaultDenom, suite.TestAccs[1].String()))
	suite.Require().NoError(err)
	queryRes, err = suite.queryClient.DenomAuthorityMetadata(suite.Ctx.Context(), &types.QueryDenomAuthorityMetadataRequest{
		Denom: suite.defaultDenom,
	})
	suite.Require().NoError(err)
	suite.Require().Equal(suite.TestAccs[1].String(), queryRes.AuthorityMetadata.Admin)

	// Make sure old admin can no longer do actions
	_, err = suite.msgServer.Burn(sdk.WrapSDKContext(suite.Ctx), types.NewMsgBurn(suite.TestAccs[0].String(), sdk.NewInt64Coin(suite.defaultDenom, 5)))
	suite.Require().Error(err)

	// Make sure the new admin works
	_, err = suite.msgServer.Mint(sdk.WrapSDKContext(suite.Ctx), types.NewMsgMint(suite.TestAccs[1].String(), sdk.NewInt64Coin(suite.defaultDenom, 5)))
	addr1bal += 5
	suite.Require().NoError(err)
	suite.Require().True(bankKeeper.GetBalance(suite.Ctx, suite.TestAccs[1], suite.defaultDenom).Amount.Int64() == addr1bal)

	// Try setting admin to empty
	_, err = suite.msgServer.ChangeAdmin(sdk.WrapSDKContext(suite.Ctx), types.NewMsgChangeAdmin(suite.TestAccs[1].String(), suite.defaultDenom, ""))
	suite.Require().NoError(err)
	queryRes, err = suite.queryClient.DenomAuthorityMetadata(suite.Ctx.Context(), &types.QueryDenomAuthorityMetadataRequest{
		Denom: suite.defaultDenom,
	})
	suite.Require().NoError(err)
	suite.Require().Equal("", queryRes.AuthorityMetadata.Admin)
}

// TestMintDenom ensures the following properties of the MintMessage:
// * Noone can mint tokens for a denom that doesn't exist
// * Only the admin of a denom can mint tokens for it
// * The admin of a denom can mint tokens for it
func (suite *KeeperTestSuite) TestMintDenom() {
	balances := make(map[string]int64)
	for _, acc := range suite.TestAccs {
		balances[acc.String()] = 0
	}

	// Create a denom
	suite.CreateDefaultDenom()

	for _, tc := range []struct {
		desc       string
		mintMsg    types.MsgMint
		expectPass bool
	}{
		{
			desc: "denom does not exist",
			mintMsg: *types.NewMsgMint(
				suite.TestAccs[0].String(),
				sdk.NewInt64Coin("factory/osmo1t7egva48prqmzl59x5ngv4zx0dtrwewc9m7z44/evmos", 10),
			),
			expectPass: false,
		},
		{
			desc: "mint is not by the admin",
			mintMsg: *types.NewMsgMintTo(
				suite.TestAccs[1].String(),
				sdk.NewInt64Coin(suite.defaultDenom, 10),
				suite.TestAccs[0].String(),
			),
			expectPass: false,
		},
		{
			desc: "success case - mint to self",
			mintMsg: *types.NewMsgMint(
				suite.TestAccs[0].String(),
				sdk.NewInt64Coin(suite.defaultDenom, 10),
			),
			expectPass: true,
		},
		{
			desc: "success case - mint to another address",
			mintMsg: *types.NewMsgMintTo(
				suite.TestAccs[0].String(),
				sdk.NewInt64Coin(suite.defaultDenom, 10),
				suite.TestAccs[1].String(),
			),
			expectPass: true,
		},
	} {
		suite.Run(fmt.Sprintf("Case %s", tc.desc), func() {
			_, err := suite.msgServer.Mint(sdk.WrapSDKContext(suite.Ctx), &tc.mintMsg)
			if tc.expectPass {
				suite.Require().NoError(err)
				balances[tc.mintMsg.MintToAddress] += tc.mintMsg.Amount.Amount.Int64()
			} else {
				suite.Require().Error(err)
			}

			mintToAddr, _ := sdk.AccAddressFromBech32(tc.mintMsg.MintToAddress)
			bal := suite.App.BankKeeper.GetBalance(suite.Ctx, mintToAddr, suite.defaultDenom).Amount
			suite.Require().Equal(bal.Int64(), balances[tc.mintMsg.MintToAddress])
		})
	}
}

func (suite *KeeperTestSuite) TestBurnDenom() {
	// Create a denom.
	suite.CreateDefaultDenom()

	// mint 1000 default token for all testAccs
	balances := make(map[string]int64)
	for _, acc := range suite.TestAccs {
		_, err := suite.msgServer.Mint(sdk.WrapSDKContext(suite.Ctx), types.NewMsgMintTo(suite.TestAccs[0].String(), sdk.NewInt64Coin(suite.defaultDenom, 1000), acc.String()))
		suite.Require().NoError(err)
		balances[acc.String()] = 1000
	}

	for _, tc := range []struct {
		desc       string
		burnMsg    types.MsgBurn
		expectPass bool
	}{
		{
			desc: "denom does not exist",
			burnMsg: *types.NewMsgBurn(
				suite.TestAccs[0].String(),
				sdk.NewInt64Coin("factory/osmo1t7egva48prqmzl59x5ngv4zx0dtrwewc9m7z44/evmos", 10),
			),
			expectPass: false,
		},
		{
			desc: "burn is not by the admin",
			burnMsg: *types.NewMsgBurnFrom(
				suite.TestAccs[1].String(),
				sdk.NewInt64Coin(suite.defaultDenom, 10),
				suite.TestAccs[0].String(),
			),
			expectPass: false,
		},
		{
			desc: "burn more than balance",
			burnMsg: *types.NewMsgBurn(
				suite.TestAccs[0].String(),
				sdk.NewInt64Coin(suite.defaultDenom, 10000),
			),
			expectPass: false,
		},
		{
			desc: "success case - burn from self",
			burnMsg: *types.NewMsgBurn(
				suite.TestAccs[0].String(),
				sdk.NewInt64Coin(suite.defaultDenom, 10),
			),
			expectPass: true,
		},
		/*{
			desc: "success case - burn from another address",
			burnMsg: *types.NewMsgBurnFrom(
				suite.TestAccs[0].String(),
				sdk.NewInt64Coin(suite.defaultDenom, 10),
				suite.TestAccs[1].String(),
			),
			expectPass: true,
		},*/
	} {
		suite.Run(fmt.Sprintf("Case %s", tc.desc), func() {
			_, err := suite.msgServer.Burn(sdk.WrapSDKContext(suite.Ctx), &tc.burnMsg)
			if tc.expectPass {
				suite.Require().NoError(err)
				balances[tc.burnMsg.BurnFromAddress] -= tc.burnMsg.Amount.Amount.Int64()
			} else {
				suite.Require().Error(err)
			}

			burnFromAddr, _ := sdk.AccAddressFromBech32(tc.burnMsg.BurnFromAddress)
			bal := suite.App.BankKeeper.GetBalance(suite.Ctx, burnFromAddr, suite.defaultDenom).Amount
			suite.Require().Equal(bal.Int64(), balances[tc.burnMsg.BurnFromAddress])
		})
	}
}

// Capability is disabled
/*func (suite *KeeperTestSuite) TestForceTransferDenom() {
	// Create a denom.
	suite.CreateDefaultDenom()

	// mint 1000 default token for all testAccs
	balances := make(map[string]int64)
	for _, acc := range suite.TestAccs {
		_, err := suite.msgServer.Mint(sdk.WrapSDKContext(suite.Ctx), types.NewMsgMintTo(suite.TestAccs[0].String(), sdk.NewInt64Coin(suite.defaultDenom, 1000), acc.String()))
		suite.Require().NoError(err)
		balances[acc.String()] = 1000
	}

	for _, tc := range []struct {
		desc             string
		forceTransferMsg types.MsgForceTransfer
		expectPass       bool
	}{
		{
			desc: "valid force transfer",
			forceTransferMsg: *types.NewMsgForceTransfer(
				suite.TestAccs[0].String(),
				sdk.NewInt64Coin(suite.defaultDenom, 10),
				suite.TestAccs[1].String(),
				suite.TestAccs[2].String(),
			),
			expectPass: true,
		},
		{
			desc: "denom does not exist",
			forceTransferMsg: *types.NewMsgForceTransfer(
				suite.TestAccs[0].String(),
				sdk.NewInt64Coin("factory/osmo1t7egva48prqmzl59x5ngv4zx0dtrwewc9m7z44/evmos", 10),
				suite.TestAccs[1].String(),
				suite.TestAccs[2].String(),
			),
			expectPass: false,
		},
		{
			desc: "forceTransfer is not by the admin",
			forceTransferMsg: *types.NewMsgForceTransfer(
				suite.TestAccs[1].String(),
				sdk.NewInt64Coin(suite.defaultDenom, 10),
				suite.TestAccs[1].String(),
				suite.TestAccs[2].String(),
			),
			expectPass: false,
		},
		{
			desc: "forceTransfer is greater than the balance of",
			forceTransferMsg: *types.NewMsgForceTransfer(
				suite.TestAccs[0].String(),
				sdk.NewInt64Coin(suite.defaultDenom, 10000),
				suite.TestAccs[1].String(),
				suite.TestAccs[2].String(),
			),
			expectPass: false,
		},
	} {
		suite.Run(fmt.Sprintf("Case %s", tc.desc), func() {
			_, err := suite.msgServer.ForceTransfer(sdk.WrapSDKContext(suite.Ctx), &tc.forceTransferMsg)
			if tc.expectPass {
				suite.Require().NoError(err)

				balances[tc.forceTransferMsg.TransferFromAddress] -= tc.forceTransferMsg.Amount.Amount.Int64()
				balances[tc.forceTransferMsg.TransferToAddress] += tc.forceTransferMsg.Amount.Amount.Int64()
			} else {
				suite.Require().Error(err)
			}

			fromAddr, err := sdk.AccAddressFromBech32(tc.forceTransferMsg.TransferFromAddress)
			suite.Require().NoError(err)
			fromBal := suite.App.BankKeeper.GetBalance(suite.Ctx, fromAddr, suite.defaultDenom).Amount
			suite.Require().True(fromBal.Int64() == balances[tc.forceTransferMsg.TransferFromAddress])

			toAddr, err := sdk.AccAddressFromBech32(tc.forceTransferMsg.TransferToAddress)
			suite.Require().NoError(err)
			toBal := suite.App.BankKeeper.GetBalance(suite.Ctx, toAddr, suite.defaultDenom).Amount
			suite.Require().True(toBal.Int64() == balances[tc.forceTransferMsg.TransferToAddress])
		})
	}
}*/

func (suite *KeeperTestSuite) TestChangeAdminDenom() {
	for _, tc := range []struct {
		desc                    string
		msgChangeAdmin          func(denom string) *types.MsgChangeAdmin
		expectedChangeAdminPass bool
		expectedAdminIndex      int
		msgMint                 func(denom string) *types.MsgMint
		expectedMintPass        bool
	}{
		{
			desc: "creator admin can't mint after setting to '' ",
			msgChangeAdmin: func(denom string) *types.MsgChangeAdmin {
				return types.NewMsgChangeAdmin(suite.TestAccs[0].String(), denom, "")
			},
			expectedChangeAdminPass: true,
			expectedAdminIndex:      -1,
			msgMint: func(denom string) *types.MsgMint {
				return types.NewMsgMint(suite.TestAccs[0].String(), sdk.NewInt64Coin(denom, 5))
			},
			expectedMintPass: false,
		},
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
			msgMint: func(denom string) *types.MsgMint {
				return types.NewMsgMint(suite.TestAccs[1].String(), sdk.NewInt64Coin(denom, 5))
			},
			expectedMintPass: true,
		},
	} {
		suite.Run(fmt.Sprintf("Case %s", tc.desc), func() {
			// setup test
			suite.SetupTest()

			// Create a denom and mint
			res, err := suite.msgServer.CreateDenom(sdk.WrapSDKContext(suite.Ctx), types.NewMsgCreateDenom(suite.TestAccs[0].String(), "bitcoin"))
			suite.Require().NoError(err)

			testDenom := res.GetNewTokenDenom()

			_, err = suite.msgServer.Mint(sdk.WrapSDKContext(suite.Ctx), types.NewMsgMint(suite.TestAccs[0].String(), sdk.NewInt64Coin(testDenom, 10)))
			suite.Require().NoError(err)

			_, err = suite.msgServer.ChangeAdmin(sdk.WrapSDKContext(suite.Ctx), tc.msgChangeAdmin(testDenom))
			if tc.expectedChangeAdminPass {
				suite.Require().NoError(err)
			} else {
				suite.Require().Error(err)
			}

			queryRes, err := suite.queryClient.DenomAuthorityMetadata(suite.Ctx.Context(), &types.QueryDenomAuthorityMetadataRequest{
				Denom: testDenom,
			})
			suite.Require().NoError(err)

			// expectedAdminIndex with negative value is assumed as admin with value of ""
			const emptyStringAdminIndexFlag = -1
			if tc.expectedAdminIndex == emptyStringAdminIndexFlag {
				suite.Require().Equal("", queryRes.AuthorityMetadata.Admin)
			} else {
				suite.Require().Equal(suite.TestAccs[tc.expectedAdminIndex].String(), queryRes.AuthorityMetadata.Admin)
			}

			// we test mint to test if admin authority is performed properly after admin change.
			if tc.msgMint != nil {
				_, err := suite.msgServer.Mint(sdk.WrapSDKContext(suite.Ctx), tc.msgMint(testDenom))
				if tc.expectedMintPass {
					suite.Require().NoError(err)
				} else {
					suite.Require().Error(err)
				}
			}
		})
	}
}

// Capability is diabled
/*func (suite *KeeperTestSuite) TestSetDenomMetaData() {
	// setup test
	suite.SetupTest()
	suite.CreateDefaultDenom()

	for _, tc := range []struct {
		desc                string
		msgSetDenomMetadata types.MsgSetDenomMetadata
		expectedPass        bool
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
			expectedPass: true,
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
		{
			desc: "non-factory denom",
			msgSetDenomMetadata: *types.NewMsgSetDenomMetadata(suite.TestAccs[0].String(), banktypes.Metadata{
				Description: "yeehaw",
				DenomUnits: []*banktypes.DenomUnit{
					{
						Denom:    "uosmo",
						Exponent: 0,
					},
					{
						Denom:    "uosmoo",
						Exponent: 6,
					},
				},
				Base:    "uosmo",
				Display: "uosmoo",
				Name:    "OSMO",
				Symbol:  "OSMO",
			}),
			expectedPass: false,
		},
		{
			desc: "wrong admin",
			msgSetDenomMetadata: *types.NewMsgSetDenomMetadata(suite.TestAccs[1].String(), banktypes.Metadata{
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
			expectedPass: false,
		},
		{
			desc: "invalid metadata (missing display denom unit)",
			msgSetDenomMetadata: *types.NewMsgSetDenomMetadata(suite.TestAccs[0].String(), banktypes.Metadata{
				Description: "yeehaw",
				DenomUnits: []*banktypes.DenomUnit{
					{
						Denom:    suite.defaultDenom,
						Exponent: 0,
					},
				},
				Base:    suite.defaultDenom,
				Display: "uosmo",
				Name:    "OSMO",
				Symbol:  "OSMO",
			}),
			expectedPass: false,
		},
	} {
		suite.Run(fmt.Sprintf("Case %s", tc.desc), func() {
			bankKeeper := suite.App.BankKeeper
			res, err := suite.msgServer.SetDenomMetadata(sdk.WrapSDKContext(suite.Ctx), &tc.msgSetDenomMetadata)
			if tc.expectedPass {
				suite.Require().NoError(err)
				suite.Require().NotNil(res)

				md, found := bankKeeper.GetDenomMetaData(suite.Ctx, suite.defaultDenom)
				suite.Require().True(found)
				suite.Require().Equal(tc.msgSetDenomMetadata.Metadata.Name, md.Name)
			} else {
				suite.Require().Error(err)
			}
		})
	}
}*/
