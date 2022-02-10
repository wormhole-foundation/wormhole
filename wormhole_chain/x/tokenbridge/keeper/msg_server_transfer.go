package keeper

import (
	"bytes"
	"context"
	"encoding/binary"
	"github.com/certusone/wormhole-chain/x/tokenbridge/types"
	whtypes "github.com/certusone/wormhole-chain/x/wormhole/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	types2 "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/holiman/uint256"
	"math"
	"math/big"
)

func (k msgServer) Transfer(goCtx context.Context, msg *types.MsgTransfer) (*types.MsgTransferResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	wormholeConfig, ok := k.wormholeKeeper.GetConfig(ctx)
	if !ok {
		return nil, whtypes.ErrNoConfig
	}

	userAcc, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}

	meta, found := k.bankKeeper.GetDenomMetaData(ctx, msg.Amount.Denom)
	if !found {
		return nil, types.ErrNoDenomMetadata
	}

	// The display denom should have the most common decimal places
	var displayDenom *types2.DenomUnit
	for _, denom := range meta.DenomUnits {
		if denom.Denom == meta.Display {
			displayDenom = denom
			break
		}
	}
	if displayDenom == nil {
		return nil, types.ErrDisplayUnitNotFound
	}
	if displayDenom.Exponent > math.MaxUint8 {
		return nil, types.ErrExponentTooLarge
	}
	decimals := uint8(displayDenom.Exponent)

	// Collect coins in module account
	// TODO: why not burn?
	err = k.bankKeeper.SendCoins(ctx, userAcc, k.accountKeeper.GetModuleAddress(types.ModuleName), sdk.Coins{msg.Amount})
	if err != nil {
		return nil, err
	}

	// Parse fees
	feeBig, ok := new(big.Int).SetString(msg.Fee, 10)
	if !ok {
		panic("invalid fee")
	}

	bridgeBalance := new(big.Int).Set(k.bankKeeper.GetBalance(ctx, k.accountKeeper.GetModuleAddress(types.ModuleName), msg.Amount.Denom).Amount.BigInt())
	truncAmount := new(big.Int).Set(msg.Amount.Amount.BigInt())
	truncFees := new(big.Int).Set(feeBig)

	// Truncate if local decimals are > 8
	if decimals > 8 {
		truncAmount = truncAmount.Div(truncAmount, new(big.Int).SetInt64(int64(math.Pow10(int(decimals-8)))))
		bridgeBalance = bridgeBalance.Div(bridgeBalance, new(big.Int).SetInt64(int64(math.Pow10(int(decimals-8)))))
		truncFees = truncFees.Div(truncFees, new(big.Int).SetInt64(int64(math.Pow10(int(decimals-8)))))
	}

	if !truncAmount.IsUint64() || !bridgeBalance.IsUint64() {
		return nil, types.ErrAmountTooHigh
	}

	// Check that the total outflow of this asset does not exceed u64
	if new(big.Int).Add(truncAmount, bridgeBalance).IsUint64() {
		return nil, types.ErrAmountTooHigh
	}

	buf := new(bytes.Buffer)
	// PayloadID
	buf.WriteByte(1)
	// Amount
	tokenAmount, ok := uint256.FromBig(truncAmount)
	if !ok {
		return nil, types.ErrInvalidAmount
	}
	buf.Write(tokenAmount.Bytes())
	// TokenAddress
	denomBytes, err := PadStringToByte32(msg.Amount.Denom)
	if err != nil {
		return nil, types.ErrDenomTooLong
	}
	buf.Write(denomBytes)
	// TokenChain
	MustWrite(buf, binary.BigEndian, uint16(wormholeConfig.ChainId))
	// To
	buf.Write(msg.ToAddress)
	// ToChain
	if msg.ToChain > math.MaxUint8 {
		return nil, types.ErrChainIDTooLarge
	}
	MustWrite(buf, binary.BigEndian, uint16(msg.ToChain))
	// Fee
	fee, ok := uint256.FromBig(truncFees)
	if !ok {
		return nil, types.ErrInvalidFee
	}
	buf.Write(fee.Bytes())

	// Check that the amount is sufficient to cover the fee
	if truncAmount.Cmp(truncFees) != 1 {
		return nil, types.ErrFeeTooHigh
	}

	// Post message
	err = k.wormholeKeeper.PostMessage(ctx, k.accountKeeper.GetModuleAddress(types.ModuleName), 0, buf.Bytes())
	if err != nil {
		return nil, err
	}

	return &types.MsgTransferResponse{}, nil
}
