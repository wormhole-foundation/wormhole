package keeper

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"math/big"
	"strings"

	"github.com/certusone/wormhole-chain/x/wormhole/keeper"
	whtypes "github.com/certusone/wormhole-chain/x/wormhole/types"
	btypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/certusone/wormhole-chain/x/tokenbridge/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type PayloadID uint8

var (
	PayloadIDTransfer  PayloadID = 1
	PayloadIDAssetMeta PayloadID = 2
)

func (k msgServer) ExecuteVAA(goCtx context.Context, msg *types.MsgExecuteVAA) (*types.MsgExecuteVAAResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Parse VAA
	v, err := keeper.ParseVAA(msg.Vaa)
	if err != nil {
		return nil, err
	}

	// Verify VAA
	err = k.wormholeKeeper.VerifyVAA(ctx, v)
	if err != nil {
		return nil, err
	}

	wormholeConfig, ok := k.wormholeKeeper.GetConfig(ctx)
	if !ok {
		return nil, whtypes.ErrNoConfig
	}

	// Replay protection
	_, known := k.GetReplayProtection(ctx, v.HexDigest())
	if known {
		return nil, types.ErrVAAAlreadyExecuted
	}

	// Check if emitter is a registered chain
	registration, found := k.GetChainRegistration(ctx, uint16(v.EmitterChain))
	if !found {
		return nil, types.ErrUnregisteredEmitter
	}
	if !bytes.Equal(v.EmitterAddress[:], registration.EmitterAddress) {
		return nil, types.ErrUnregisteredEmitter
	}

	if len(v.Payload) < 1 {
		return nil, types.ErrVAAPayloadInvalid
	}

	payloadID := PayloadID(v.Payload[0])
	payload := v.Payload[1:]

	switch payloadID {
	case PayloadIDTransfer:
		if len(payload) != 132 {
			return nil, types.ErrVAAPayloadInvalid
		}
		amount := new(big.Int).SetBytes(payload[:32])
		tokenAddress := payload[32:64]
		tokenChain := binary.BigEndian.Uint16(payload[64:66])
		to := payload[66:98]
		toChain := binary.BigEndian.Uint16(payload[98:100])
		fee := new(big.Int).SetBytes(payload[100:132])

		// Check that the transfer is to this chain
		if uint32(toChain) != wormholeConfig.ChainId {
			return nil, types.ErrInvalidTargetChain
		}

		identifier := ""
		if uint32(tokenChain) != wormholeConfig.ChainId {
			// Mint new wrapped assets if the coin is from another chain
			identifier = "b" + GetCoinIdentifier(tokenChain, tokenAddress)

			err = k.bankKeeper.MintCoins(ctx, types.ModuleName, sdk.Coins{
				{
					Denom:  identifier,
					Amount: sdk.NewIntFromBigInt(amount),
				},
			})
			if err != nil {
				return nil, err
			}
		} else {
			// Recover the coin denom from the token address if it's a native coin
			identifier = strings.TrimLeft(string(tokenAddress), "\x00")
		}

		meta, found := k.bankKeeper.GetDenomMetaData(ctx, identifier)
		if !found && uint32(tokenChain) != wormholeConfig.ChainId {
			return nil, types.ErrAssetNotRegistered
		} else if !found {
			return nil, types.ErrNoDenomMetadata
		}

		// Find the display denom to figure out decimals
		var displayDenom *btypes.DenomUnit
		for _, denom := range meta.DenomUnits {
			if denom.Denom == meta.Display {
				displayDenom = denom
				break
			}
		}
		if displayDenom == nil {
			return nil, types.ErrDisplayUnitNotFound
		}

		// If the original decimals exceed 8 un-truncate the amounts
		if displayDenom.Exponent > 8 {
			amount = amount.Mul(amount, new(big.Int).SetInt64(int64(math.Pow10(int(displayDenom.Exponent-8)))))
			fee = fee.Mul(fee, new(big.Int).SetInt64(int64(math.Pow10(int(displayDenom.Exponent-8)))))
		}

		moduleAccount := k.accountKeeper.GetModuleAddress(types.ModuleName)

		// Transfer amount to recipient
		err = k.bankKeeper.SendCoins(ctx, moduleAccount, to, sdk.Coins{
			{
				Denom:  identifier,
				Amount: sdk.NewIntFromBigInt(new(big.Int).Sub(amount, fee)),
			},
		})
		if err != nil {
			return nil, err
		}

		txSender, err := sdk.AccAddressFromBech32(msg.Creator)
		if err != nil {
			panic(err)
		}
		// Transfer fee to tx sender if it is not 0
		if fee.Sign() != 0 {
			err = k.bankKeeper.SendCoins(ctx, moduleAccount, txSender, sdk.Coins{
				{
					Denom:  identifier,
					Amount: sdk.NewIntFromBigInt(fee),
				},
			})
		}

		err = ctx.EventManager().EmitTypedEvent(&types.EventTransferReceived{
			TokenChain:   uint32(tokenChain),
			TokenAddress: tokenAddress,
			To:           sdk.AccAddress(to).String(),
			FeeRecipient: txSender.String(),
			Amount:       amount.String(),
			Fee:          fee.String(),
			LocalDenom:   identifier,
		})
		if err != nil {
			panic(err)
		}

	case PayloadIDAssetMeta:
		if len(payload) != 99 {
			return nil, types.ErrVAAPayloadInvalid
		}
		tokenAddress := payload[:32]
		tokenChain := binary.BigEndian.Uint16(payload[32:34])
		decimals := payload[34]
		symbol := string(payload[35:67])
		symbol = strings.Trim(symbol, "\x00")
		name := string(payload[67:99])
		name = strings.Trim(name, "\x00")

		// Don't allow native assets to be registered as wrapped asset
		if uint32(tokenChain) == wormholeConfig.ChainId {
			return nil, types.ErrNativeAssetRegistration
		}

		identifier := GetCoinIdentifier(tokenChain, tokenAddress)
		rollBackProtection, found := k.GetCoinMetaRollbackProtection(ctx, identifier)
		if found && rollBackProtection.LastUpdateSequence >= v.Sequence {
			return nil, types.ErrAssetMetaRollback
		}

		k.bankKeeper.SetDenomMetaData(ctx, btypes.Metadata{
			Description: fmt.Sprintf("Wormhole wrapped asset from chain %d with address %x", tokenChain, tokenAddress),
			DenomUnits: []*btypes.DenomUnit{
				{
					Denom:    "b" + identifier,
					Exponent: 0,
				},
				{
					Denom:    identifier,
					Exponent: uint32(decimals),
				},
			},
			Base:    "b" + identifier,
			Display: identifier,
			Name:    name,
			Symbol:  symbol,
		})
		k.SetCoinMetaRollbackProtection(ctx, types.CoinMetaRollbackProtection{
			Index:              identifier,
			LastUpdateSequence: v.Sequence,
		})

		err = ctx.EventManager().EmitTypedEvent(&types.EventAssetRegistrationUpdate{
			TokenChain:   uint32(tokenChain),
			TokenAddress: tokenAddress,
			Name:         name,
			Symbol:       symbol,
			Decimals:     uint32(decimals),
		})
		if err != nil {
			panic(err)
		}
	default:
		return nil, types.ErrUnknownPayloadType
	}

	// Prevent replay
	k.SetReplayProtection(ctx, types.ReplayProtection{Index: v.HexDigest()})

	return &types.MsgExecuteVAAResponse{}, nil
}

func GetCoinIdentifier(tokenChain uint16, tokenAddress []byte) string {
	return fmt.Sprintf("wh/%d/%x", tokenChain, tokenAddress)
}
