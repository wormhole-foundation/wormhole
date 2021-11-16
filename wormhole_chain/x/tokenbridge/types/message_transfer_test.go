package types

import (
	"strconv"
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
				ToAddress: make([]byte, 32),
				Fee:       strconv.Itoa(0),
			},
		},
	}
	for _, tt := range tests {
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
