package keeper

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"math/big"

	"github.com/certusone/wormhole-chain/x/tokenbridge/types"
	whtypes "github.com/certusone/wormhole-chain/x/wormhole/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

func (k msgServer) Transfer(goCtx context.Context, msg *types.MsgTransfer) (*types.MsgTransferResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	msg.Amount = sdk.NormalizeCoin(msg.Amount)
	msg.Fee = sdk.NormalizeCoin(msg.Fee)

	wormholeConfig, ok := k.wormholeKeeper.GetConfig(ctx)
	if !ok {
		return nil, whtypes.ErrNoConfig
	}

	userAcc, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, err
	}

	if _, found := k.GetChainRegistration(ctx, msg.ToChain); !found {
		return nil, types.ErrInvalidTargetChain
	}

	meta, found := k.bankKeeper.GetDenomMetaData(ctx, msg.Amount.Denom)
	if !found {
		return nil, types.ErrNoDenomMetadata
	}

	moduleAddress := k.accountKeeper.GetModuleAddress(types.ModuleName)
	bridgeBalance, err := types.Truncate(k.bankKeeper.GetBalance(ctx, moduleAddress, msg.Amount.Denom), meta)
	if err != nil {
		return nil, fmt.Errorf("failed to truncate bridge balance: %w", err)
	}
	amount, err := types.Truncate(msg.Amount, meta)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", types.ErrInvalidAmount, err)
	}

	fees, err := types.Truncate(msg.Fee, meta)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", types.ErrInvalidFee, err)
	}

	if amount.IsLT(fees) {
		return nil, types.ErrFeeTooHigh
	}

	if !amount.Amount.IsUint64() || !bridgeBalance.Amount.IsUint64() {
		return nil, types.ErrAmountTooHigh
	}

	// Check that the total outflow of this asset does not exceed u64
	if !bridgeBalance.Add(amount).Amount.IsUint64() {
		return nil, types.ErrAmountTooHigh
	}

	_, _, wrapped := types.GetWrappedCoinMeta(msg.Amount.Denom)
	if wrapped {
		// We previously minted these coins so just burn them now.
		if err := k.bankKeeper.BurnCoins(ctx, types.ModuleName, sdk.Coins{amount}); err != nil {
			return nil, sdkerrors.Wrap(err, "failed to burn wrapped coins")
		}
	} else {
		// Collect coins in the module account.
		if err := k.bankKeeper.SendCoins(ctx, userAcc, moduleAddress, sdk.Coins{amount}); err != nil {
			return nil, sdkerrors.Wrap(err, "failed to send coins to module account")
		}
	}

	buf := new(bytes.Buffer)
	// PayloadID
	buf.WriteByte(1)
	// Amount
	tokenAmountBytes32 := bytes32(amount.Amount.BigInt())
	buf.Write(tokenAmountBytes32[:])
	tokenChain, tokenAddress, err := types.GetTokenMeta(wormholeConfig, msg.Amount.Denom)
	if err != nil {
		return nil, err
	}
	// TokenAddress
	buf.Write(tokenAddress[:])
	// TokenChain
	MustWrite(buf, binary.BigEndian, tokenChain)
	// To
	buf.Write(msg.ToAddress)
	// ToChain
	MustWrite(buf, binary.BigEndian, uint16(msg.ToChain))
	// Fee
	feeBytes32 := bytes32(fees.Amount.BigInt())
	buf.Write(feeBytes32[:])

	// Post message
	emitterAddress := whtypes.EmitterAddressFromAccAddress(moduleAddress)
	err = k.wormholeKeeper.PostMessage(ctx, emitterAddress, 0, buf.Bytes())
	if err != nil {
		return nil, err
	}

	return &types.MsgTransferResponse{}, nil
}

func bytes32(i *big.Int) [32]byte {
	var out [32]byte

	i.FillBytes(out[:])

	return out
}
