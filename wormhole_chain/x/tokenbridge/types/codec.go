package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgExecuteGovernanceVAA{}, "tokenbridge/ExecuteGovernanceVAA", nil)
	cdc.RegisterConcrete(&MsgExecuteVAA{}, "tokenbridge/ExecuteVAA", nil)
	cdc.RegisterConcrete(&MsgAttestToken{}, "tokenbridge/AttestToken", nil)
	cdc.RegisterConcrete(&MsgTransfer{}, "tokenbridge/Transfer", nil)
	// this line is used by starport scaffolding # 2
}

func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgExecuteGovernanceVAA{},
	)
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgExecuteVAA{},
	)
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgAttestToken{},
	)
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgTransfer{},
	)
	// this line is used by starport scaffolding # 3

	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}

var (
	ModuleCdc = codec.NewProtoCodec(cdctypes.NewInterfaceRegistry())
)
