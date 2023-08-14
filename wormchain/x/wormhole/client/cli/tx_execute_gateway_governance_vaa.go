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

// CmdExecuteGatewayGovernanceVaa will submit and execute a governance VAA for wormhole Gateway.
func CmdExecuteGatewayGovernanceVaa() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "execute-gateway-governance-vaa [vaa-hex]",
		Short: "Execute the provided Wormhole Gateway governance VAA",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			vaaBz, err := hex.DecodeString(args[0])
			if err != nil {
				return err
			}

			msg := types.MsgExecuteGatewayGovernanceVaa{
				Signer: clientCtx.GetFromAddress().String(),
				Vaa:    vaaBz,
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
