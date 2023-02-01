package cli

import (
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/spf13/cobra"
	"github.com/wormhole-foundation/wormchain/x/wormhole/types"
)

var _ = strconv.Itoa(0)

// StoreCodeCmd will upload code to be reused.
func CmdCreateAllowedAddress() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "create-allowed-address [wormchain-address] [human-readable-name-of-key]",
		Short:   "Allowlist an address to be able to submit tx to wormchain. Must be submitted by a validator account.",
		Aliases: []string{"allowlist", "allow"},
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			address := args[0]
			name := args[1]

			msg := types.MsgCreateAllowlistEntryRequest{
				Signer:  clientCtx.GetFromAddress().String(),
				Address: address,
				Name:    name,
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

// StoreCodeCmd will upload code to be reused.
func CmdDeleteAllowedAddress() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "delete-allowed-address [wormchain-address]",
		Short:   "Remove an allowlist entry. The allowlist must be stale (only valid under old guardian set) or you must be the creator of the allowlist.",
		Aliases: []string{"delete-allowlist", "remove-allowlist", "disallow"},
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			address := args[0]

			msg := types.MsgDeleteAllowlistEntryRequest{
				Signer:  clientCtx.GetFromAddress().String(),
				Address: address,
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
