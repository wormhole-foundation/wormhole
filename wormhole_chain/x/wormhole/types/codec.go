package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
	gov "github.com/cosmos/cosmos-sdk/x/gov/types"
)

func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgExecuteGovernanceVAA{}, "wormhole/ExecuteGovernanceVAA", nil)
	cdc.RegisterConcrete(&MsgRegisterAccountAsGuardian{}, "wormhole/RegisterAccountAsGuardian", nil)
	// this line is used by starport scaffolding # 2
}

func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgExecuteGovernanceVAA{},
	)
	registry.RegisterImplementations((*gov.Content)(nil),
		&GovernanceWormholeMessageProposal{},
		&GuardianSetUpdateProposal{})
	registry.RegisterImplementations((*sdk.Msg)(nil),
		&MsgRegisterAccountAsGuardian{},
	)
	// this line is used by starport scaffolding # 3

	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}

var (
	ModuleCdc = codec.NewProtoCodec(cdctypes.NewInterfaceRegistry())
)
