package types

import (
	"testing"

	"github.com/certusone/wormhole-chain/testutil/sample"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/stretchr/testify/require"
)

func TestMsgRegisterAccountAsGuardian_ValidateBasic(t *testing.T) {
	tests := []struct {
		name string
		msg  MsgRegisterAccountAsGuardian
		err  error
	}{
		{
			name: "invalid address",
			msg: MsgRegisterAccountAsGuardian{
				Signer: "invalid_address",
			},
			err: sdkerrors.ErrInvalidAddress,
		}, {
			name: "valid address",
			msg: MsgRegisterAccountAsGuardian{
				Signer: sample.AccAddress(),
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
