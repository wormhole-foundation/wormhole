package keeper

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"math"

	whtypes "github.com/certusone/wormhole-chain/x/wormhole/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/certusone/wormhole-chain/x/tokenbridge/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k msgServer) AttestToken(goCtx context.Context, msg *types.MsgAttestToken) (*types.MsgAttestTokenResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	wormholeConfig, ok := k.wormholeKeeper.GetConfig(ctx)
	if !ok {
		return nil, whtypes.ErrNoConfig
	}

	meta, found := k.bankKeeper.GetDenomMetaData(ctx, msg.Denom)
	if !found {
		return nil, types.ErrNoDenomMetadata
	}

	// Don't attest wrapped assets (including uworm)
	_, _, wrapped := types.GetWrappedCoinMeta(meta.Display)
	if wrapped {
		return nil, types.ErrAttestWormholeToken
	}

	// The display denom should have the most common decimal places
	var displayDenom *banktypes.DenomUnit
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
	exponent := uint8(displayDenom.Exponent)

	buf := new(bytes.Buffer)
	// PayloadID
	buf.WriteByte(2)
	tokenChain, tokenAddress, err := types.GetTokenMeta(wormholeConfig, meta.Base)
	if err != nil {
		return nil, err
	}
	// TokenAddress
	buf.Write(tokenAddress[:])
	// TokenChain
	MustWrite(buf, binary.BigEndian, tokenChain)
	MustWrite(buf, binary.BigEndian, uint16(wormholeConfig.ChainId))
	// Decimals
	MustWrite(buf, binary.BigEndian, exponent)
	// Symbol
	symbolBytes, err := types.PadStringToByte32(meta.Symbol)
	if err != nil {
		return nil, types.ErrSymbolTooLong
	}
	buf.Write(symbolBytes[:])
	// Name
	nameBytes, err := types.PadStringToByte32(meta.Name)
	if err != nil {
		return nil, types.ErrNameTooLong
	}
	buf.Write(nameBytes[:])

	// Post message
	moduleAddress := k.accountKeeper.GetModuleAddress(types.ModuleName)
	emitterAddress := whtypes.EmitterAddressFromAccAddress(moduleAddress)
	err = k.wormholeKeeper.PostMessage(ctx, emitterAddress, 0, buf.Bytes())
	if err != nil {
		return nil, err
	}

	return &types.MsgAttestTokenResponse{}, nil
}

// MustWrite calls binary.Write and panics on errors
func MustWrite(w io.Writer, order binary.ByteOrder, data interface{}) {
	if err := binary.Write(w, order, data); err != nil {
		panic(fmt.Errorf("failed to write binary data: %v", data).Error())
	}
}
