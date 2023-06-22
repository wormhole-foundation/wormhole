package types

import (
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type AccountKeeper interface {
	// Methods imported from account should be defined here
}

type BankKeeper interface {
	// Methods imported from bank should be defined here
}

type WasmdKeeper interface {
	// For StoreCode
	Create(ctx sdk.Context, creator sdk.AccAddress, wasmCode []byte, instantiateAccess *wasmtypes.AccessConfig) (codeID uint64, checksum []byte, err error)
	Instantiate(ctx sdk.Context, codeID uint64, creator, admin sdk.AccAddress, initMsg []byte, label string, deposit sdk.Coins) (sdk.AccAddress, []byte, error)
	Migrate(ctx sdk.Context, contractAddress sdk.AccAddress, caller sdk.AccAddress, newCodeID uint64, msg []byte) ([]byte, error)
}
