package cli

import (
	"encoding/hex"
	"strconv"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/spf13/cast"
	"github.com/spf13/cobra"
	"github.com/wormhole-foundation/wormchain/x/wormhole/types"
)

var _ = strconv.Itoa(0)

// StoreCodeCmd will upload code to be reused.
func CmdAddWasmInstantiateAllowlist() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-wasm-instantiate-allowlist [bech32 contract addr] [codeId] [vaa-hex]",
		Short: "Allowlist a contract address to be able to instantiate a specific code ID",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			address := args[0]

			codeId, err := cast.ToUint64E(args[1])
			if err != nil {
				return err
			}

			vaaBz, err := hex.DecodeString(args[2])
			if err != nil {
				return err
			}

			msg := types.MsgAddWasmInstantiateAllowlist{
				Signer:  clientCtx.GetFromAddress().String(),
				Address: address,
				CodeId:  codeId,
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

func CmdDeleteWasmInstantiateAllowlist() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "delete-wasm-instantiate-allowlist [bech32 contract addr] [codeId] [vaa-hex]",
		Short: "Delete a contract address and code ID pair from the wasm instantiate allowlist",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			address := args[0]

			codeId, err := cast.ToUint64E(args[1])
			if err != nil {
				return err
			}

			vaaBz, err := hex.DecodeString(args[2])
			if err != nil {
				return err
			}

			msg := types.MsgDeleteWasmInstantiateAllowlist{
				Signer:  clientCtx.GetFromAddress().String(),
				Address: address,
				CodeId:  codeId,
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
