package types

import (
	wasmdtypes "github.com/CosmWasm/wasmd/x/wasm/types"
)

func (msg MsgInstantiateContract) ToWasmd() wasmdtypes.MsgInstantiateContract {
	return wasmdtypes.MsgInstantiateContract{
		Sender: msg.Signer,
		CodeID: msg.CodeID,
		Label:  msg.Label,
		Msg:    msg.Msg,
	}
}

func (msg MsgStoreCode) ToWasmd() wasmdtypes.MsgStoreCode {
	return wasmdtypes.MsgStoreCode{
		Sender:       msg.Signer,
		WASMByteCode: msg.WASMByteCode,
	}
}

func (msg MsgMigrateContract) ToWasmd() wasmdtypes.MsgMigrateContract {
	return wasmdtypes.MsgMigrateContract{
		Sender:   msg.Signer,
		Msg:      msg.Msg,
		Contract: msg.Contract,
		CodeID:   msg.CodeID,
	}
}
