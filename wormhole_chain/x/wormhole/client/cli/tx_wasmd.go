package cli

import (
	"fmt"
	"io/ioutil"
	"strconv"

	"encoding/hex"

	"github.com/CosmWasm/wasmd/x/wasm/ioutils"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/cast"
	"github.com/spf13/cobra"
	"github.com/wormhole-foundation/wormhole-chain/x/wormhole/types"
)

var _ = strconv.Itoa(0)

func parseStoreCodeArgs(file string, sender sdk.AccAddress, vaa []byte) (types.MsgStoreCode, error) {
	wasm, err := ioutil.ReadFile(file)
	if err != nil {
		return types.MsgStoreCode{}, err
	}

	// gzip the wasm file
	if ioutils.IsWasm(wasm) {
		wasm, err = ioutils.GzipIt(wasm)

		if err != nil {
			return types.MsgStoreCode{}, err
		}
	} else if !ioutils.IsGzip(wasm) {
		return types.MsgStoreCode{}, fmt.Errorf("invalid input file. Use wasm binary or gzip")
	}
	return types.MsgStoreCode{
		Signer:       sender.String(),
		WASMByteCode: wasm,
		Vaa:          vaa,
	}, nil
}

// StoreCodeCmd will upload code to be reused.
func CmdStoreCode() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "store [wasm file] [vaa-hex]",
		Short:   "Upload a wasm binary with vaa",
		Aliases: []string{"upload", "st", "s"},
		Args:    cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			vaaBz, err := hex.DecodeString(args[1])
			if err != nil {
				return err
			}

			msg, err := parseStoreCodeArgs(args[0], clientCtx.GetFromAddress(), vaaBz)
			if err != nil {
				return err
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

func CmdInstantiateContract() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "instantiate [label] [code_id_int64] [json_encoded_init_args] [vaa-hex]",
		Short: "Register a guardian public key with a wormhole chain address.",
		Args:  cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}

			labelStr := args[0]

			codeId, err := cast.ToUint64E(args[1])
			if err != nil {
				return err
			}

			initMsg := args[2]

			vaaBz, err := hex.DecodeString(args[3])
			if err != nil {
				return err
			}

			msg := types.MsgInstantiateContract{
				Signer: clientCtx.GetFromAddress().String(),
				CodeID: codeId,
				Label:  labelStr,
				Msg:    []byte(initMsg),
				Vaa:    vaaBz,
			}
			if err := msg.ValidateBasic(); err != nil {
				return err
			}
			return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), &msg)
		},
	}

	cmd.Flags().String("label", "", "A human-readable name for this contract in lists")
	flags.AddTxFlagsToCmd(cmd)

	return cmd
}
