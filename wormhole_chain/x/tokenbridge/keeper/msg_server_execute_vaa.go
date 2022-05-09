package keeper

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
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
	registration, found := k.GetChainRegistration(ctx, uint32(v.EmitterChain))
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
		unnormalizedAmount := new(big.Int).SetBytes(payload[:32])
		var tokenAddress [32]byte
		copy(tokenAddress[:], payload[32:64])
		tokenChain := binary.BigEndian.Uint16(payload[64:66])
		var to [20]byte
		copy(to[:], payload[78:98])
		toChain := binary.BigEndian.Uint16(payload[98:100])
		unnormalizedFee := new(big.Int).SetBytes(payload[100:132])

		// Check that the transfer is to this chain
		if uint32(toChain) != wormholeConfig.ChainId {
			return nil, types.ErrInvalidTargetChain
		}

		identifier := ""
		var wrapped bool
		if types.IsWORMToken(tokenChain, tokenAddress) {
			identifier = "uworm"
			// We mint wormhole tokens because they are not native to wormhole chain
			wrapped = true
		} else if uint32(tokenChain) != wormholeConfig.ChainId {
			// Mint new wrapped assets if the coin is from another chain
			identifier = "b" + types.GetWrappedCoinIdentifier(tokenChain, tokenAddress)
			wrapped = true
		} else {
			// Recover the coin denom from the token address if it's a native coin
			identifier = strings.TrimLeft(string(tokenAddress[:]), "\x00")
			wrapped = false
		}

		meta, found := k.bankKeeper.GetDenomMetaData(ctx, identifier)
		if !found {
			if !wrapped {
				return nil, types.ErrNoDenomMetadata
			} else {
				return nil, types.ErrAssetNotRegistered
			}
		}

		amount, err := types.Untruncate(unnormalizedAmount, meta)
		if err != nil {
			return nil, err
		}

		fee, err := types.Untruncate(unnormalizedFee, meta)
		if err != nil {
			return nil, err
		}

		if wrapped {
			err = k.bankKeeper.MintCoins(ctx, types.ModuleName, sdk.Coins{
				{
					Denom:  identifier,
					Amount: sdk.NewIntFromBigInt(amount),
				},
			})
			if err != nil {
				return nil, err
			}
		}

		moduleAccount := k.accountKeeper.GetModuleAddress(types.ModuleName)

		amtLessFees := sdk.Coins{
			{
				Denom:  identifier,
				Amount: sdk.NewIntFromBigInt(new(big.Int).Sub(amount, fee)),
			},
		}

		err = k.bankKeeper.SendCoins(ctx, moduleAccount, to[:], amtLessFees)
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
			if err != nil {
				panic(err)
			}
		}

		err = ctx.EventManager().EmitTypedEvent(&types.EventTransferReceived{
			TokenChain:   uint32(tokenChain),
			TokenAddress: tokenAddress[:],
			To:           sdk.AccAddress(to[:]).String(),
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
		var tokenAddress [32]byte
		copy(tokenAddress[:], payload[:32])
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

		if types.IsWORMToken(tokenChain, tokenAddress) {
			return nil, types.ErrNativeAssetRegistration
		}

		identifier := types.GetWrappedCoinIdentifier(tokenChain, tokenAddress)
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
			TokenAddress: tokenAddress[:],
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
