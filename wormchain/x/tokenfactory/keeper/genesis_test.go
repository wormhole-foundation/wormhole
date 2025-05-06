package keeper_test

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/wormhole-foundation/wormchain/x/tokenfactory/types"
)

func (suite *KeeperTestSuite) TestGenesis() {
	genesisState := types.GenesisState{
		FactoryDenoms: []types.GenesisDenom{
			{
				Denom: "factory/wormhole13p05zcjlfsxsjua77es6g9kxg8kr243nrhf7jg/bitcoin",
				AuthorityMetadata: types.DenomAuthorityMetadata{
					Admin: "wormhole13p05zcjlfsxsjua77es6g9kxg8kr243nrhf7jg",
				},
			},
			{
				Denom: "factory/wormhole13p05zcjlfsxsjua77es6g9kxg8kr243nrhf7jg/diff-admin",
				AuthorityMetadata: types.DenomAuthorityMetadata{
					Admin: "wormhole13p05zcjlfsxsjua77es6g9kxg8kr243nrhf7jg",
				},
			},
			{
				Denom: "factory/wormhole13p05zcjlfsxsjua77es6g9kxg8kr243nrhf7jg/litecoin",
				AuthorityMetadata: types.DenomAuthorityMetadata{
					Admin: "wormhole13p05zcjlfsxsjua77es6g9kxg8kr243nrhf7jg",
				},
			},
		},
	}

	suite.SetupTestForInitGenesis()
	wormchain := suite.App

	// Test both with bank denom metadata set, and not set.
	for i, denom := range genesisState.FactoryDenoms {
		// hacky, sets bank metadata to exist if i != 0, to cover both cases.
		if i != 0 {
			wormchain.BankKeeper.SetDenomMetaData(suite.Ctx, banktypes.Metadata{Base: denom.GetDenom()})
		}
	}

	if err := wormchain.TokenFactoryKeeper.SetParams(suite.Ctx, types.Params{DenomCreationFee: sdk.Coins{sdk.NewInt64Coin("stake", 100)}}); err != nil {
		panic(err)
	}
	wormchain.TokenFactoryKeeper.InitGenesis(suite.Ctx, genesisState)

	// check that the module account is now initialized
	tokenfactoryModuleAccount := wormchain.AccountKeeper.GetAccount(suite.Ctx, wormchain.AccountKeeper.GetModuleAddress(types.ModuleName))
	suite.Require().NotNil(tokenfactoryModuleAccount)

	exportedGenesis := wormchain.TokenFactoryKeeper.ExportGenesis(suite.Ctx)
	suite.Require().NotNil(exportedGenesis)
	suite.Require().Equal(genesisState, *exportedGenesis)
}
