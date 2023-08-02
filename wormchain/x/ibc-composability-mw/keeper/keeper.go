package keeper

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/wormhole-foundation/wormchain/x/ibc-composability-mw/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	transfertypes "github.com/cosmos/ibc-go/v4/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v4/modules/core/04-channel/types"
	ibcexported "github.com/cosmos/ibc-go/v4/modules/core/exported"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wormholekeeper "github.com/wormhole-foundation/wormchain/x/wormhole/keeper"
)

type Keeper struct {
	cdc            codec.BinaryCodec
	storeKey       storetypes.StoreKey
	wasmKeeper     *wasmkeeper.Keeper
	wormholeKeeper *wormholekeeper.Keeper

	retriesOnTimeout uint8
	forwardTimeout   time.Duration
}

func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey storetypes.StoreKey,
	wasmKeeper *wasmkeeper.Keeper,
	wormholeKeeper *wormholekeeper.Keeper,
	retriesOnTimeout uint8,
	forwardTimeout time.Duration,
) *Keeper {
	return &Keeper{
		cdc:              cdc,
		storeKey:         storeKey,
		wasmKeeper:       wasmKeeper,
		wormholeKeeper:   wormholeKeeper,
		retriesOnTimeout: retriesOnTimeout,
		forwardTimeout:   forwardTimeout,
	}
}

func (k *Keeper) SetWasmKeeper(wasmkeeper *wasmkeeper.Keeper) {
	k.wasmKeeper = wasmkeeper
}

// OnRecvPacket checks the memo field on this packet and if the memo indicates this packet
// should be handled by the ibc composability middleware, it updates the memo according to the payload
func (k Keeper) OnRecvPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
) (channeltypes.Packet, ibcexported.Acknowledgement) {
	// If wasm keeper is not set, bypass this middleware
	if k.wasmKeeper == nil {
		return packet, nil
	}

	var data transfertypes.FungibleTokenPacketData
	if err := transfertypes.ModuleCdc.UnmarshalJSON(packet.GetData(), &data); err != nil {
		return packet, channeltypes.NewErrorAcknowledgement(err)
	}

	memo := make(map[string]interface{})
	err := json.Unmarshal([]byte(data.Memo), &memo)
	if err != nil || memo["gateway_ibc_token_bridge_payload"] == nil {
		// not a packet that can be parsed
		return packet, channeltypes.NewErrorAcknowledgement(fmt.Errorf("ibc-composability-mw: must be a valid memo for gateway"))
	}

	parsedPayload, err := types.VerifyAndParseGatewayPayload(data.Memo)
	if err != nil {
		return packet, channeltypes.NewErrorAcknowledgement(err)
	}

	// Get ibc translator contract address
	ibcTranslatorContract := k.wormholeKeeper.GetIbcComposabilityMwContract(ctx)

	// Look up chain id's channel
	req := types.IbcTranslatorQueryMsg{
		IbcChannel: types.QueryIbcChannel{
			ChainID: parsedPayload.ChainId,
		},
	}
	reqBz, err := json.Marshal(req)
	if err != nil {
		return packet, channeltypes.NewErrorAcknowledgement(err)
	}
	ibcTranslatorAddr, err := sdk.AccAddressFromBech32(ibcTranslatorContract.ContractAddress)
	if err != nil {
		return packet, channeltypes.NewErrorAcknowledgement(err)
	}
	resp, err := k.wasmKeeper.QuerySmart(ctx, ibcTranslatorAddr, reqBz)

	var newMemo string
	isNewMemoPfm := false
	if err == nil {
		isNewMemoPfm = true
		// If response exists, create PFM memo
		newMemo, err = types.FormatPfmMemo(parsedPayload, resp, k.forwardTimeout, k.retriesOnTimeout)
		if err != nil {
			return packet, channeltypes.NewErrorAcknowledgement(err)
		}
	} else {
		// If response doesn't exist, create ibc-hooks memo
		newMemo, err = types.FormatIbcHooksMemo(parsedPayload, ibcTranslatorContract.ContractAddress)
		if err != nil {
			return packet, channeltypes.NewErrorAcknowledgement(err)
		}
	}

	data.Memo = newMemo
	newData, err := transfertypes.ModuleCdc.MarshalJSON(&data)
	if err != nil {
		return packet, channeltypes.NewErrorAcknowledgement(err)
	}

	if isNewMemoPfm {
		// Store orginal data to save while packet is in flight (PFM-only)
		key := types.TransposedDataKey(packet.DestinationChannel, packet.DestinationPort, packet.Sequence)
		store := ctx.KVStore(k.storeKey)
		store.Set(key, packet.GetData())
	}

	packet.Data = newData

	return packet, nil
}

func (k Keeper) GetAndClearTransposedData(
	ctx sdk.Context,
	channel string,
	port string,
	sequence uint64,
) []byte {
	store := ctx.KVStore(k.storeKey)
	key := types.TransposedDataKey(channel, port, sequence)
	if !store.Has(key) {
		return nil
	}

	data := store.Get(key)
	store.Delete(key)

	return data
}

func (k *Keeper) WriteAcknowledgement(ctx sdk.Context, packet ibcexported.PacketI) ibcexported.PacketI {
	transposedData := k.GetAndClearTransposedData(ctx, packet.GetDestChannel(), packet.GetDestPort(), packet.GetSequence())
	if transposedData != nil {
		concretePacket, ok := packet.(channeltypes.Packet)
		if ok {
			concretePacket.Data = transposedData
			packet = concretePacket
		}
	}
	return packet
}
