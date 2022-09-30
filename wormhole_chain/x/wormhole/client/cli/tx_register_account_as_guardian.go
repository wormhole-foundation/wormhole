package cli

import (
	"fmt"
	"strconv"

	"encoding/hex"
	"encoding/json"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/spf13/cobra"
	"github.com/wormhole-foundation/wormhole-chain/x/wormhole/types"
)

var _ = strconv.Itoa(0)

func CmdRegisterAccountAsGuardian() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "register-account-as-guardian [guardian-pubkey] [signature]",
		Short: "Register a guardian public key with a wormhole chain address.",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			argGuardianPubkey := new(types.GuardianKey)
			err = json.Unmarshal([]byte(args[0]), argGuardianPubkey)
			if err != nil {
				return err
			}
			argSignature, err := hex.DecodeString(args[1])
			if err != nil {
				return fmt.Errorf("malformed signature: %w", err)
			}
			fmt.Println("PUBKEY:", hex.EncodeToString(argGuardianPubkey.Key))

			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			msg := types.NewMsgRegisterAccountAsGuardian(
				clientCtx.GetFromAddress().String(),
				argGuardianPubkey,
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
