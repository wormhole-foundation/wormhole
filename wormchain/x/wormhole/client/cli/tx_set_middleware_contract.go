package cli

import (
	"encoding/hex"
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/spf13/cobra"
	"github.com/wormhole-foundation/wormchain/x/wormhole/types"
)

var _ = strconv.Itoa(0)

// CmdSetMiddlewareContract will set the contract that wormhole's middleware will use.
func CmdSetMiddlewareContract() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set-middleware-contract [bech32 contract addr] [vaa-hex]",
		Short: "Sets the contract that wormhole's middleware will use",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			address := args[0]

			vaaBz, err := hex.DecodeString(args[1])
			if err != nil {
				return err
			}

			msg := types.MsgSetWormholeMiddlewareContract{
				Signer:  clientCtx.GetFromAddress().String(),
				Address: address,
				Vaa:     vaaBz,
			}

			if err = msg.ValidateBasic(); err != nil {
				return err
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), &msg)
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}