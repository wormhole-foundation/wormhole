package types

import (
	"testing"

	"github.com/stretchr/testify/suite"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type CodecTestSuite struct {
	suite.Suite
}

func TestCodecSuite(t *testing.T) {
	suite.Run(t, new(CodecTestSuite))
}

func (suite *CodecTestSuite) TestRegisterInterfaces() {
	registry := codectypes.NewInterfaceRegistry()
	registry.RegisterInterface(sdk.MsgInterfaceProtoName, (*sdk.Msg)(nil))
	RegisterInterfaces(registry)

	impls := registry.ListImplementations(sdk.MsgInterfaceProtoName)
	suite.Require().Equal(7, len(impls))
	suite.Require().ElementsMatch([]string{
		"/osmosis.tokenfactory.v1beta1.MsgCreateDenom",
		"/osmosis.tokenfactory.v1beta1.MsgMint",
		"/osmosis.tokenfactory.v1beta1.MsgBurn",
		"/osmosis.tokenfactory.v1beta1.MsgChangeAdmin",
		"/osmosis.tokenfactory.v1beta1.MsgSetDenomMetadata",
		"/osmosis.tokenfactory.v1beta1.MsgForceTransfer",
		"/osmosis.tokenfactory.v1beta1.MsgUpdateParams",
	}, impls)
}
