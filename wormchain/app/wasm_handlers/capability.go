package wasm_handlers

import (
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitykeeper "github.com/cosmos/cosmos-sdk/x/capability/keeper"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"
)

type CapabilityKeeperHandler struct {
	ScopedKeeper capabilitykeeper.ScopedKeeper
}

var _ wasmtypes.CapabilityKeeper = &CapabilityKeeperHandler{}

func (b *CapabilityKeeperHandler) GetCapability(ctx sdk.Context, name string) (*capabilitytypes.Capability, bool) {
	return b.ScopedKeeper.GetCapability(ctx, name)
}
func (b *CapabilityKeeperHandler) ClaimCapability(ctx sdk.Context, cap *capabilitytypes.Capability, name string) error {
	return b.ScopedKeeper.ClaimCapability(ctx, cap, name)
}
func (b *CapabilityKeeperHandler) AuthenticateCapability(ctx sdk.Context, cap *capabilitytypes.Capability, name string) bool {
	return b.ScopedKeeper.AuthenticateCapability(ctx, cap, name)
}
