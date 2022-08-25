package types

import (
	"testing"

	"github.com/certusone/wormhole-chain/testutil/sample"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"
)

func TestMsgTransfer_ValidateBasic(t *testing.T) {
	tests := []struct {
		name string
		msg  MsgTransfer
		err  error
	}{
		{
			name: "invalid address",
			msg: MsgTransfer{
				Creator: "invalid_address",
			},
			err: sdkerrors.ErrInvalidAddress,
		}, {
			name: "valid address",
			msg: MsgTransfer{
				Creator:   sample.AccAddress(),
				Amount:    sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(10)),
				ToChain:   1,
				ToAddress: make([]byte, 32),
				Fee:       sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(1)),
			},
		}, {
			name: "negative amount",
			msg: MsgTransfer{
				Creator:   sample.AccAddress(),
				Amount:    sdk.Coin{Denom: sdk.DefaultBondDenom, Amount: sdk.NewInt(-10)},
				ToChain:   0,
				ToAddress: make([]byte, 32),
				Fee:       sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(0)),
			},
			err: ErrInvalidAmount,
		}, {
			name: "invalid amount denom",
			msg: MsgTransfer{
				Creator:   sample.AccAddress(),
				Amount:    sdk.Coin{Denom: "007test", Amount: sdk.NewInt(10)},
				ToChain:   0,
				ToAddress: make([]byte, 32),
				Fee:       sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(0)),
			},
			err: ErrInvalidAmount,
		}, {
			name: "negative fee",
			msg: MsgTransfer{
				Creator:   sample.AccAddress(),
				Amount:    sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(0)),
				ToChain:   0,
				ToAddress: make([]byte, 32),
				Fee:       sdk.Coin{Denom: sdk.DefaultBondDenom, Amount: sdk.NewInt(-10)},
			},
			err: ErrInvalidFee,
		}, {
			name: "invalid fee denom",
			msg: MsgTransfer{
				Creator:   sample.AccAddress(),
				Amount:    sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(0)),
				ToChain:   0,
				ToAddress: make([]byte, 32),
				Fee:       sdk.Coin{Denom: "007test", Amount: sdk.NewInt(10)},
			},
			err: ErrInvalidFee,
		}, {
			name: "invalid target chain",
			msg: MsgTransfer{
				Creator:   sample.AccAddress(),
				Amount:    sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(10)),
				ToChain:   uint32(^uint16(0)) + 1,
				ToAddress: make([]byte, 32),
				Fee:       sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(1)),
			},
			err: ErrInvalidTargetChain,
		}, {
			name: "invalid target address",
			msg: MsgTransfer{
				Creator:   sample.AccAddress(),
				Amount:    sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(10)),
				ToChain:   1,
				ToAddress: make([]byte, 16),
				Fee:       sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(1)),
			},
			err: ErrInvalidToAddress,
		}, {
			name: "mismatched denoms",
			msg: MsgTransfer{
				Creator:   sample.AccAddress(),
				Amount:    sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(10)),
				ToChain:   1,
				ToAddress: make([]byte, 32),
				Fee:       sdk.NewCoin("test", sdk.NewInt(1)),
			},
			err: ErrInvalidFee,
		}, {
			name: "fee too high",
			msg: MsgTransfer{
				Creator:   sample.AccAddress(),
				Amount:    sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(10)),
				ToChain:   1,
				ToAddress: make([]byte, 32),
				Fee:       sdk.NewCoin(sdk.DefaultBondDenom, sdk.NewInt(10)),
			},
			err: ErrFeeTooHigh,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.ValidateBasic()
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
				return
			}
			require.NoError(t, err)
		})
	}
}
