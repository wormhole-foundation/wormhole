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
	"github.com/wormhole-foundation/wormchain/x/wormhole/types"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"golang.org/x/crypto/sha3"
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
		Short:   "Upload a wasm binary with vaa, or just compute the hash for vaa if [vaa-hex] is omitted",
		Aliases: []string{"upload", "st", "s"},
		Args:    cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			hash_only := len(args) == 1
			vaaBz := []byte{}
			if !hash_only {
				vaaBz, err = hex.DecodeString(args[1])
				if err != nil {
					return err
				}
			}

			msg, err := parseStoreCodeArgs(args[0], clientCtx.GetFromAddress(), vaaBz)
			if err != nil {
				return err
			}

			if !hash_only {
				if err = msg.ValidateBasic(); err != nil {
					return err
				}
				return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), &msg)
			} else {
				var hashWasm [32]byte
				keccak := sha3.NewLegacyKeccak256()
				keccak.Write(msg.WASMByteCode)
				keccak.Sum(hashWasm[:0])
				fmt.Println(hex.EncodeToString(hashWasm[:]))
				return nil
			}
		},
	}

	flags.AddTxFlagsToCmd(cmd)
	return cmd
}

func CmdInstantiateContract() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "instantiate [label] [code_id_int64] [json_encoded_init_args] [vaa-hex]",
		Short: "Instantiate a wasmd contract, or just compute the hash for vaa if vaa is omitted",
		Args:  cobra.RangeArgs(3, 4),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			hash_only := len(args) == 3

			labelStr := args[0]

			codeId, err := cast.ToUint64E(args[1])
			if err != nil {
				return err
			}

			initMsg := args[2]

			vaaBz := []byte{}
			if !hash_only {
				vaaBz, err = hex.DecodeString(args[3])
				if err != nil {
					return err
				}
			}

			msg := types.MsgInstantiateContract{
				Signer: clientCtx.GetFromAddress().String(),
				CodeID: codeId,
				Label:  labelStr,
				Msg:    []byte(initMsg),
				Vaa:    vaaBz,
			}
			if !hash_only {
				if err := msg.ValidateBasic(); err != nil {
					return err
				}
				return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), &msg)
			} else {
				hash := vaa.CreateInstatiateCosmwasmContractHash(msg.CodeID, msg.Label, msg.Msg)
				fmt.Println(hex.EncodeToString(hash[:]))
				return nil
			}
		},
	}

	cmd.Flags().String("label", "", "A human-readable name for this contract in lists")
	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

func CmdMigrateContract() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "migrate [contract] [code_id_uint64] [json_encoded_init_args] [vaa-hex]",
		Short: "Migrate a wasmd contract, or just compute the hash for vaa if vaa is omitted",
		Args:  cobra.RangeArgs(3, 4),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			hash_only := len(args) == 3

			contract := args[0]

			codeId, err := cast.ToUint64E(args[1])
			if err != nil {
				return err
			}

			initMsg := args[2]

			vaaBz := []byte{}
			if !hash_only {
				vaaBz, err = hex.DecodeString(args[3])
				if err != nil {
					return err
				}
			}

			msg := types.MsgMigrateContract{
				Signer:   clientCtx.GetFromAddress().String(),
				CodeID:   codeId,
				Contract: contract,
				Msg:      []byte(initMsg),
				Vaa:      vaaBz,
			}
			if hash_only {
				hash := vaa.CreateMigrateCosmwasmContractHash(msg.CodeID, msg.Contract, msg.Msg)
				fmt.Println(hex.EncodeToString(hash[:]))
				return nil
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
