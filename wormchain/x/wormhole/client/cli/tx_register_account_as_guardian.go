package cli

import (
	"fmt"
	"strconv"

	"encoding/hex"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/spf13/cobra"
	"github.com/wormhole-foundation/wormchain/x/wormhole/types"
)

var _ = strconv.Itoa(0)

func CmdRegisterAccountAsGuardian() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "register-account-as-guardian [signature]",
		Short: "Register a guardian public key with a wormhole chain address.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			argSignature, err := hex.DecodeString(args[0])
			if err != nil {
				return fmt.Errorf("malformed signature: %w", err)
			}

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgRegisterAccountAsGuardian(
				clientCtx.GetFromAddress().String(),
				argSignature,
			)
			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}
