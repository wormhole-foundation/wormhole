package keeper

import (
	"bytes"
	"context"
	"encoding/binary"
	"math/big"

	"github.com/certusone/wormhole-chain/x/tokenbridge/types"
	whtypes "github.com/certusone/wormhole-chain/x/wormhole/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/holiman/uint256"
)

func (k msgServer) Transfer(goCtx context.Context, msg *types.MsgTransfer) (*types.MsgTransferResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	wormholeConfig, ok := k.wormholeKeeper.GetConfig(ctx)
	if !ok {
		return nil, whtypes.ErrNoConfig
	}

	userAcc, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return nil, err
	}

	meta, found := k.bankKeeper.GetDenomMetaData(ctx, msg.Amount.Denom)
	if !found {
		return nil, types.ErrNoDenomMetadata
	}

	moduleAddress := k.accountKeeper.GetModuleAddress(types.ModuleName)
	_, _, wrapped := types.GetWrappedCoinMeta(msg.Amount.Denom)
	if wrapped {
		// We previously minted these coins so just burn them now.
		if err := k.bankKeeper.BurnCoins(ctx, types.ModuleName, sdk.Coins{msg.Amount}); err != nil {
			return nil, sdkerrors.Wrap(err, "failed to burn wrapped coins")
		}
	} else {
		// Collect coins in the module account.
		if err := k.bankKeeper.SendCoins(ctx, userAcc, moduleAddress, sdk.Coins{msg.Amount}); err != nil {
			return nil, sdkerrors.Wrap(err, "failed to send coins to module account")
		}
	}

	// Parse fees
	feeBig, ok := new(big.Int).SetString(msg.Fee, 10)
	if !ok || feeBig.Sign() == -1 {
		return nil, types.ErrInvalidFee
	}

	bridgeBalance := new(big.Int).Set(k.bankKeeper.GetBalance(ctx, k.accountKeeper.GetModuleAddress(types.ModuleName), msg.Amount.Denom).Amount.BigInt())
	amount := new(big.Int).Set(msg.Amount.Amount.BigInt())
	fees := new(big.Int).Set(feeBig)

	truncAmount, err := types.Truncate(amount, meta)
	if err != nil {
		return nil, err
	}

	truncFees, err := types.Truncate(fees, meta)
	if err != nil {
		return nil, err
	}

	truncBridgeBalance, err := types.Truncate(bridgeBalance, meta)
	if err != nil {
		return nil, err
	}

	if !truncAmount.IsUint64() || !bridgeBalance.IsUint64() {
		return nil, types.ErrAmountTooHigh
	}

	// Check that the total outflow of this asset does not exceed u64
	if !new(big.Int).Add(truncAmount, truncBridgeBalance).IsUint64() {
		return nil, types.ErrAmountTooHigh
	}

	buf := new(bytes.Buffer)
	// PayloadID
	buf.WriteByte(1)
	// Amount
	tokenAmount, overflow := uint256.FromBig(truncAmount)
	if overflow {
		return nil, types.ErrInvalidAmount
	}
	tokenAmountBytes32 := tokenAmount.Bytes32()
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
	fee, overflow := uint256.FromBig(truncFees)
	if overflow {
		return nil, types.ErrInvalidFee
	}
	feeBytes32 := fee.Bytes32()
	buf.Write(feeBytes32[:])

	// Check that the amount is sufficient to cover the fee
	if truncAmount.Cmp(truncFees) != 1 {
		return nil, types.ErrFeeTooHigh
	}

	// Post message
	emitterAddress := whtypes.EmitterAddressFromAccAddress(moduleAddress)
	err = k.wormholeKeeper.PostMessage(ctx, emitterAddress, 0, buf.Bytes())
	if err != nil {
		return nil, err
	}

	return &types.MsgTransferResponse{}, nil
}
