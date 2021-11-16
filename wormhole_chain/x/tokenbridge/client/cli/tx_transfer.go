package cli

import (
	"encoding/hex"
	"fmt"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"math/big"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/certusone/wormhole-chain/x/tokenbridge/types"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
)

var _ = strconv.Itoa(0)

func CmdTransfer() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "transfer [amount] [to_chain] [to_address] [fee]",
		Short: "Broadcast message Transfer",
		Args:  cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			coins, err := sdk.ParseCoinNormalized(args[0])
			if err != nil {
				return err
			}

			chainID, err := strconv.ParseUint(args[1], 10, 16)
			if err != nil {
				return err
			}

			toAddress, err := hex.DecodeString(args[2])
			if err != nil {
				return fmt.Errorf("to address invalid: %w", err)
			}

			fee, ok := new(big.Int).SetString(args[3], 10)
			if !ok {
				return fmt.Errorf("invalid fee (must be a number)")
			}

			msg := types.NewMsgTransfer(
				clientCtx.GetFromAddress().String(),
				coins,
				uint16(chainID),
				toAddress,
				fee,
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
