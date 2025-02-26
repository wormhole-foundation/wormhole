package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
	authzcodec "github.com/cosmos/cosmos-sdk/x/authz/codec"
)

var (
	amino = codec.NewLegacyAmino()

	// ModuleCdc references the global erc20 module codec. Note, the codec should
	// ONLY be used in certain instances of tests and for JSON encoding.
	//
	// The actual codec used for serialization should be provided to modules/erc20 and
	// defined at the application level.
	ModuleCdc = codec.NewProtoCodec(codectypes.NewInterfaceRegistry())

	// AminoCdc is a amino codec created to support amino JSON compatible msgs.
	AminoCdc = codec.NewAminoCodec(amino)
)

const (
	// Amino names
	createTFDenom        = "osmosis/tokenfactory/create-denom"
	mintTFDenom          = "osmosis/tokenfactory/mint"
	burnTFDenom          = "osmosis/tokenfactory/burn"
	forceTransferTFDenom = "osmosis/tokenfactory/force-transfer"
	changeAdminTFDenom   = "osmosis/tokenfactory/change-admin"
	updateTFparams       = "osmosis/tokenfactory/msg-update-params"
)

// NOTE: This is required for the GetSignBytes function
func init() {
	RegisterLegacyAminoCodec(amino)

	sdk.RegisterLegacyAminoCodec(amino)
	// cryptocodec.RegisterCrypto(amino)
	// codec.RegisterEvidences(amino)

	// Register all Amino interfaces and concrete types on the authz Amino codec
	// so that this can later be used to properly serialize MsgGrant and MsgExec
	// instances.
	RegisterLegacyAminoCodec(authzcodec.Amino)

	amino.Seal()
}

func RegisterInterfaces(registry codectypes.InterfaceRegistry) {
	registry.RegisterImplementations(
		(*sdk.Msg)(nil),
		&MsgCreateDenom{},
		&MsgMint{},
		&MsgBurn{},
		&MsgForceTransfer{},
		&MsgChangeAdmin{},
		&MsgUpdateParams{},
	)
	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}

func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgCreateDenom{}, createTFDenom, nil)
	cdc.RegisterConcrete(&MsgMint{}, mintTFDenom, nil)
	cdc.RegisterConcrete(&MsgBurn{}, burnTFDenom, nil)
	cdc.RegisterConcrete(&MsgForceTransfer{}, forceTransferTFDenom, nil)
	cdc.RegisterConcrete(&MsgChangeAdmin{}, changeAdminTFDenom, nil)
	cdc.RegisterConcrete(&MsgUpdateParams{}, updateTFparams, nil)
}
