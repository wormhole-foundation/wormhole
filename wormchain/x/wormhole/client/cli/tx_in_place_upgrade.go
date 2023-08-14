package cli

import (
	"encoding/hex"
	"errors"
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/spf13/cobra"
	"github.com/wormhole-foundation/wormchain/x/wormhole/types"
)

var _ = strconv.Itoa(0)

// CmdSetIbcComposabilityMwContract will set the contract that ibc composability middleware will use.
func CmdRunInPlaceUpgrade() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run-in-place-upgrade [type] [vaa-hex]",
		Short: "Runs the in place upgrade specified by the provided governance VAA",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.MsgInPlaceUpgrade{
				Signer: clientCtx.GetFromAddress().String(),
			}

			vaaBz, err := hex.DecodeString(args[1])
			if err != nil {
				return err
			}

			// Set the in place upgrade type based on the [type] argument
			upgradeType := args[0]
			switch upgradeType {
			case "set_tokenfactory_pfm_default_params":
				msg.SetTokenfactoryPfmDefaultParams = &types.MsgSetTokenFactoryPfmDefaultParams{
					Vaa: vaaBz,
				}
			default:
				return errors.New("unrecognized in place upgrade type")
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
