package cli

import (
	"crypto/ecdsa"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/cosmos/cosmos-sdk/client/flags"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/cobra"

	"github.com/cosmos/cosmos-sdk/client"
	// "github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/wormhole-foundation/wormchain/x/wormhole/types"
	wormholesdk "github.com/wormhole-foundation/wormhole/sdk"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

// normal vaa flags
const FLAG_KEY = "key"
const FLAG_EMITTER_CHAIN = "emitter-chain"
const FLAG_INDEX = "index"
const FLAG_SEQUENCE = "sequence"
const FLAG_NONCE = "nonce"
const FLAG_PAYLOAD = "payload"

// governance vaa flags
const FLAG_MODULE = "module"
const FLAG_ACTION = "action"
const FLAG_CHAIN = "chain"
const FLAG_PUBLIC_KEY = "public-key"
const FLAG_NEXT_INDEX = "next-index"

// GetGenesisCmd returns the genesis related commands for this module
func GetGenesisCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                        "genesis",
		Short:                      fmt.Sprintf("%s genesis subcommands", types.ModuleName),
		DisableFlagParsing:         true,
		SuggestionsMinimumDistance: 2,
		RunE:                       client.ValidateCmd,
	}

	cmd.AddCommand(CmdGenerateTestGuardianKey())
	cmd.AddCommand(CmdDecodeAddress())
	cmd.AddCommand(CmdGenerateVaa())
	cmd.AddCommand(CmdGenerateGovernanceVaa())
	cmd.AddCommand(CmdGenerateGuardianSetUpdatea())
	cmd.AddCommand(CmdTestSignAddress())

	return cmd
}

func CmdGenerateTestGuardianKey() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate-test-guardian-keypair [output-private-key.hex] [address.hex]",
		Short: "Generate a guardian keypair for testing use",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			outPrivatePath := args[0]
			outPublicPath := args[1]

			// https://ethereum.org/en/developers/docs/accounts/#account-creation)

			key, err := crypto.GenerateKey()
			if err != nil {
				return err
			}
			addr := crypto.PubkeyToAddress(key.PublicKey)
			private_key := [32]byte{}
			key.D.FillBytes(private_key[:])

			err = ioutil.WriteFile(outPrivatePath, []byte(hex.EncodeToString(private_key[:])), 0644)
			if err != nil {
				return err
			}
			ioutil.WriteFile(outPublicPath, []byte(hex.EncodeToString(addr.Bytes())), 0644)
			if err != nil {
				return err
			}
			return nil
		},
	}

	return cmd
}

func CmdDecodeAddress() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "decode-address [address]",
		Short: "Decode an address from either account, validator, or evm format",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			addrString := args[0]
			if strings.HasPrefix(addrString, sdk.GetConfig().GetBech32AccountAddrPrefix()) {
				addr, err := sdk.AccAddressFromBech32(addrString)
				if err != nil {
					return nil
				}
				fmt.Println(base64.StdEncoding.EncodeToString(addr))
			} else if strings.HasPrefix(addrString, sdk.GetConfig().GetBech32ValidatorAddrPrefix()) {
				addr, err := sdk.AccAddressFromBech32(addrString)
				if err != nil {
					return nil
				}
				fmt.Println(base64.StdEncoding.EncodeToString(addr))
			} else {
				// treat as hex
				addr, err := hex.DecodeString(strings.TrimPrefix(addrString, "0x"))
				if err != nil {
					return err
				}
				fmt.Println(base64.StdEncoding.EncodeToString(addr))
			}

			return nil
		},
	}

	return cmd
}

func ImportKeyFromFile(filePath string) (*ecdsa.PrivateKey, error) {
	priv := ecdsa.PrivateKey{}
	bz, err := ioutil.ReadFile(filePath)
	if err != nil {
		return &priv, err
	}
	return ImportKeyFromHex(string(bz))
}

func ImportPublicKeyFromFile(filePath string) ([]byte, error) {
	hexBz, err := ioutil.ReadFile(filePath)
	if err != nil {
		return []byte{}, err
	}
	hexStr := string(hexBz)
	bz, err := hex.DecodeString(hexStr)
	if err != nil {
		return []byte{}, err
	}
	return bz, nil
}

func ImportKeyFromHex(privHex string) (*ecdsa.PrivateKey, error) {
	priv := ecdsa.PrivateKey{}
	priv_bz, err := hex.DecodeString(privHex)
	if err != nil {
		return &priv, err
	}
	k := big.NewInt(0)
	k.SetBytes(priv_bz)

	priv.PublicKey.Curve = crypto.S256()
	priv.D = k
	priv.PublicKey.X, priv.PublicKey.Y = priv.PublicKey.Curve.ScalarBaseMult(k.Bytes())
	return &priv, nil
}

func parseVaaFromFlags(cmd *cobra.Command) (vaa.VAA, error) {
	var GOVERNANCE_EMITTER = [32]byte{00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 04}

	emitterChain, err := cmd.Flags().GetUint16(FLAG_EMITTER_CHAIN)
	if err != nil {
		return vaa.VAA{}, err
	}
	index, err := cmd.Flags().GetUint32(FLAG_INDEX)
	if err != nil {
		return vaa.VAA{}, err
	}
	nonce, err := cmd.Flags().GetUint32(FLAG_NONCE)
	if err != nil {
		return vaa.VAA{}, err
	}
	seq, err := cmd.Flags().GetUint64(FLAG_SEQUENCE)
	if err != nil {
		return vaa.VAA{}, err
	}
	payloadHex, err := cmd.Flags().GetString(FLAG_PAYLOAD)
	if err != nil {
		return vaa.VAA{}, err
	}
	payload, err := hex.DecodeString(payloadHex)
	if err != nil {
		return vaa.VAA{}, err
	}

	v := vaa.VAA{
		Version:          uint8(1),
		GuardianSetIndex: index,
		Signatures:       nil,
		Timestamp:        time.Now(),
		Nonce:            nonce,
		Sequence:         seq,
		ConsistencyLevel: uint8(32),
		EmitterChain:     vaa.ChainID(emitterChain),
		EmitterAddress:   vaa.Address(GOVERNANCE_EMITTER),
		Payload:          payload,
	}
	return v, nil

}

func addVaaFlags(cmd *cobra.Command) {
	cmd.Flags().StringArray(FLAG_KEY, []string{}, "guardian private key file(s) to sign with (hex format) in order.")
	cmd.Flags().Uint16(FLAG_EMITTER_CHAIN, 0, "emitter chain")
	cmd.Flags().Uint32(FLAG_INDEX, 0, "guardian set index")
	cmd.Flags().Uint64(FLAG_SEQUENCE, 0, "sequence number")
	cmd.Flags().Uint32(FLAG_NONCE, 0, "nonce")
	cmd.Flags().String(FLAG_PAYLOAD, "", "payload (hex format)")
}
func addGovVaaFlags(cmd *cobra.Command) {
	cmd.Flags().String(FLAG_MODULE, "", "module (ascii string)")
	cmd.Flags().Uint8(FLAG_ACTION, 0, "action")
	cmd.Flags().Uint16(FLAG_CHAIN, 0, "chain")
}

func CmdGenerateVaa() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate-vaa",
		Short: "generate and sign a vaa with any payload",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) (err error) {

			privateKeys := []*ecdsa.PrivateKey{}
			privateKeysFiles, err := cmd.Flags().GetStringArray(FLAG_KEY)
			if err != nil {
				return err
			}
			for _, privFile := range privateKeysFiles {
				priv, err := ImportKeyFromFile(privFile)
				if err != nil {
					return err
				}
				privateKeys = append(privateKeys, priv)
			}
			v, err := parseVaaFromFlags(cmd)
			if err != nil {
				return err
			}
			for i, key := range privateKeys {
				v.AddSignature(key, uint8(i))
			}

			v_bz, err := v.Marshal()
			if err != nil {
				return err
			}
			fmt.Println(hex.EncodeToString(v_bz))

			return nil
		},
	}
	addVaaFlags(cmd)

	return cmd
}

func CmdGenerateGovernanceVaa() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate-gov-vaa",
		Short: "generate and sign a governance vaa with any payload",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) (err error) {

			privateKeys := []*ecdsa.PrivateKey{}
			privateKeysFiles, err := cmd.Flags().GetStringArray(FLAG_KEY)
			if err != nil {
				return err
			}
			for _, privFile := range privateKeysFiles {
				priv, err := ImportKeyFromFile(privFile)
				if err != nil {
					return err
				}
				privateKeys = append(privateKeys, priv)
			}
			v, err := parseVaaFromFlags(cmd)
			if err != nil {
				return err
			}
			gov_payload := v.Payload
			moduleString, err := cmd.Flags().GetString(FLAG_MODULE)
			if err != nil {
				return err
			}
			action, err := cmd.Flags().GetUint8(FLAG_ACTION)
			if err != nil {
				return err
			}
			chain, err := cmd.Flags().GetUint16(FLAG_CHAIN)
			if err != nil {
				return err
			}
			module := [32]byte{}
			copy(module[len(module)-len(moduleString):], []byte(moduleString))
			msg := types.NewGovernanceMessage(module, action, chain, gov_payload)
			v.Payload = msg.MarshalBinary()
			v.EmitterChain = 1

			for i, key := range privateKeys {
				v.AddSignature(key, uint8(i))
				// address := crypto.PubkeyToAddress(privateKeys[0].PublicKey)
				// fmt.Println("signed using ", hex.EncodeToString(address[:]))
			}

			v_bz, err := v.Marshal()
			if err != nil {
				return err
			}
			fmt.Println(hex.EncodeToString(v_bz))

			return nil
		},
	}

	addVaaFlags(cmd)
	addGovVaaFlags(cmd)

	return cmd
}

func CmdGenerateGuardianSetUpdatea() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "generate-guardian-update-vaa",
		Short: "generate and sign a governance vaa with any payload",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) (err error) {

			privateKeys := []*ecdsa.PrivateKey{}
			privateKeysFiles, err := cmd.Flags().GetStringArray(FLAG_KEY)
			if err != nil {
				return err
			}
			for _, privFile := range privateKeysFiles {
				priv, err := ImportKeyFromFile(privFile)
				if err != nil {
					return err
				}
				privateKeys = append(privateKeys, priv)
			}
			v, err := parseVaaFromFlags(cmd)
			if err != nil {
				return err
			}
			next_index, err := cmd.Flags().GetUint32(FLAG_NEXT_INDEX)
			if err != nil {
				return err
			}

			publicKeys := [][]byte{}
			pubKeysFiles, err := cmd.Flags().GetStringArray(FLAG_PUBLIC_KEY)
			if err != nil {
				return err
			}
			for _, pubFile := range pubKeysFiles {
				pubBz, err := ImportPublicKeyFromFile(pubFile)
				if err != nil {
					return err
				}
				publicKeys = append(publicKeys, pubBz)
			}
			set_update := make([]byte, 4)
			binary.BigEndian.PutUint32(set_update, next_index)
			set_update = append(set_update, uint8(len(pubKeysFiles)))
			// Add keys to set_update
			for _, pubkey := range publicKeys {
				set_update = append(set_update, pubkey...)
			}

			action := vaa.ActionGuardianSetUpdate
			chain := 3104
			module := [32]byte{}
			copy(module[:], vaa.CoreModule)
			msg := types.NewGovernanceMessage(module, byte(action), uint16(chain), set_update)
			v.Payload = msg.MarshalBinary()
			v.EmitterChain = 1

			for i, key := range privateKeys {
				v.AddSignature(key, uint8(i))
			}

			v_bz, err := v.Marshal()
			if err != nil {
				return err
			}
			fmt.Println(hex.EncodeToString(v_bz))

			return nil
		},
	}

	addVaaFlags(cmd)
	cmd.Flags().StringArray(FLAG_PUBLIC_KEY, []string{}, "guardian public key file(s) to include in new set (hex/evm format) in order.")
	cmd.Flags().Uint32(FLAG_NEXT_INDEX, 0, "next guardian set index")

	return cmd
}

func CmdTestSignAddress() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "test-sign-address",
		Short: "Test method sign the validator address to use for registering as a guardian.  Use guardiand for production, not this method.  Read guardian key as hex in $GUARDIAN_KEY env variable. use --from to indicate address to sign.",
		Args:  cobra.ExactArgs(0),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			clientCtx, err := client.GetClientTxContext(cmd)
			if err != nil {
				return err
			}
			info, err := clientCtx.Keyring.Key(clientCtx.From)
			if err != nil {
				return err
			}

			keyHex := os.Getenv("GUARDIAN_KEY")
			key, err := ImportKeyFromHex(keyHex)
			if err != nil {
				return err
			}
			addr := info.GetAddress()
			addrHash := crypto.Keccak256Hash(wormholesdk.SignedWormchainAddressPrefix, addr)
			sig, err := crypto.Sign(addrHash[:], key)
			if err != nil {
				return err
			}
			fmt.Println(hex.EncodeToString(sig))

			return nil
		},
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}
