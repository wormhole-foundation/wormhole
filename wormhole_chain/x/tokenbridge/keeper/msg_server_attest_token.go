package keeper

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	whtypes "github.com/certusone/wormhole-chain/x/wormhole/types"
	types2 "github.com/cosmos/cosmos-sdk/x/bank/types"
	"io"
	"math"
	"strings"

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

	// Detect wormhole wrapped assets
	if strings.Index(meta.Display, "wh/") == 0 {
		return nil, types.ErrAttestWormholeToken
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

	buf := new(bytes.Buffer)
	// PayloadID
	buf.WriteByte(2)
	// TokenAddress
	denomBytes, err := PadStringToByte32(meta.Base)
	if err != nil {
		return nil, types.ErrDenomTooLong
	}
	buf.Write(denomBytes)
	// TokenChain
	MustWrite(buf, binary.BigEndian, uint16(wormholeConfig.ChainId))
	// Decimals
	MustWrite(buf, binary.BigEndian, uint8(displayDenom.Exponent))
	// Symbol
	symbolBytes, err := PadStringToByte32(meta.Symbol)
	if err != nil {
		return nil, types.ErrSymbolTooLong
	}
	buf.Write(symbolBytes)
	// Name
	nameBytes, err := PadStringToByte32(meta.Name)
	if err != nil {
		return nil, types.ErrNameTooLong
	}
	buf.Write(nameBytes)

	// Post message
	err = k.wormholeKeeper.PostMessage(ctx, k.accountKeeper.GetModuleAddress(types.ModuleName), 0, buf.Bytes())
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

// PadStringToByte32 left zero pads a string to the ethereum type bytes32
func PadStringToByte32(s string) ([]byte, error) {
	if len(s) > 32 {
		return nil, fmt.Errorf("string is too long; %d > 32", len(s))
	}

	out := make([]byte, 32)
	for i := 32 - len(s); i < 32; i++ {
		out[i] = s[i-(32-len(s))]
	}

	return out, nil
}
