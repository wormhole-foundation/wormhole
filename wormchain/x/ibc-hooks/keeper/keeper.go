package keeper

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/types/address"

	"github.com/tendermint/tendermint/libs/log"

	"github.com/wormhole-foundation/wormchain/x/ibc-hooks/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type (
	Keeper struct {
		storeKey sdk.StoreKey
	}
)

// NewKeeper returns a new instance of the x/ibchooks keeper
func NewKeeper(
	storeKey sdk.StoreKey,
) Keeper {
	return Keeper{
		storeKey: storeKey,
	}
}

// Logger returns a logger for the x/tokenfactory module
func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}

func GetPacketKey(channel string, packetSequence uint64) []byte {
	return []byte(fmt.Sprintf("%s::%d", channel, packetSequence))
}

// StorePacketCallback stores which contract will be listening for the ack or timeout of a packet
func (k Keeper) StorePacketCallback(ctx sdk.Context, channel string, packetSequence uint64, contract string) {
	store := ctx.KVStore(k.storeKey)
	store.Set(GetPacketKey(channel, packetSequence), []byte(contract))
}

// GetPacketCallback returns the bech32 addr of the contract that is expecting a callback from a packet
func (k Keeper) GetPacketCallback(ctx sdk.Context, channel string, packetSequence uint64) string {
	store := ctx.KVStore(k.storeKey)
	return string(store.Get(GetPacketKey(channel, packetSequence)))
}

// DeletePacketCallback deletes the callback from storage once it has been processed
func (k Keeper) DeletePacketCallback(ctx sdk.Context, channel string, packetSequence uint64) {
	store := ctx.KVStore(k.storeKey)
	store.Delete(GetPacketKey(channel, packetSequence))
}

func DeriveIntermediateSender(channel, originalSender, bech32Prefix string) (string, error) {
	senderStr := fmt.Sprintf("%s/%s", channel, originalSender)
	senderHash32 := address.Hash(types.SenderPrefix, []byte(senderStr))
	sender := sdk.AccAddress(senderHash32[:])
	return sdk.Bech32ifyAddressBytes(bech32Prefix, sender)
}
