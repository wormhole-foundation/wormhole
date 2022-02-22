package main

import (
	"context"
	"fmt"
	"github.com/certusone/wormhole/node/pkg/ethereum/abi"
	"github.com/certusone/wormhole/node/pkg/vaa"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/spf13/cobra"
	"math"
	"os"
	"strconv"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "eth",
	Short: "Wormhole Ethereum Client",
}

var governanceVAACommand = &cobra.Command{
	Use:   "execute_governance [VAA]",
	Short: "Execute a governance VAA",
	Run:   executeGovernance,
	Args:  cobra.ExactArgs(1),
}

var postMessageCommand = &cobra.Command{
	Use:   "post_message [NONCE] [NUM_CONFIRMATIONS] [MESSAGE]",
	Short: "Post a message to wormhole",
	Run:   postMessage,
	Args:  cobra.ExactArgs(3),
}

var (
	contractAddress string
	rpcUrl          string
	key             string
)

func init() {
	rootCmd.PersistentFlags().StringVar(&contractAddress, "contract", "", "Address of the Wormhole contract")
	rootCmd.PersistentFlags().StringVar(&rpcUrl, "rpc", "", "Ethereum RPC address")
	rootCmd.PersistentFlags().StringVar(&key, "key", "", "Key to sign the transaction with (hex-encoded)")
	rootCmd.AddCommand(governanceVAACommand)
	rootCmd.AddCommand(postMessageCommand)
}

func postMessage(cmd *cobra.Command, args []string) {
	nonce, err := strconv.ParseUint(args[0], 10, 32)
	if err != nil {
		cmd.PrintErrln("Could not parse nonce", err)
		os.Exit(1)
	}

	consistencyLevel, err := strconv.ParseUint(args[1], 10, 8)
	if err != nil {
		cmd.PrintErrln("Could not parse confirmation number", err)
		os.Exit(1)
	}

	message := common.Hex2Bytes(args[2])

	ethC, err := getEthClient()
	if err != nil {
		cmd.PrintErrln(err)
		os.Exit(1)
	}

	signer, addr, err := getSigner(ethC)
	if err != nil {
		cmd.PrintErrln(err)
		os.Exit(1)
	}

	t, err := getTransactor(ethC)
	if err != nil {
		cmd.PrintErrln(err)
		os.Exit(1)
	}

	res, err := t.PublishMessage(&bind.TransactOpts{
		From:   addr,
		Signer: signer,
	}, uint32(nonce), message, uint8(consistencyLevel))
	if err != nil {
		cmd.PrintErrln(err)
		os.Exit(1)
	}

	println("Posted tx. Hash:", res.Hash().String())
}

func executeGovernance(cmd *cobra.Command, args []string) {
	ethC, err := getEthClient()
	if err != nil {
		cmd.PrintErrln(err)
		os.Exit(1)
	}

	signer, addr, err := getSigner(ethC)
	if err != nil {
		cmd.PrintErrln(err)
		os.Exit(1)
	}

	t, err := getTransactor(ethC)
	if err != nil {
		cmd.PrintErrln(err)
		os.Exit(1)
	}

	vaaData := common.Hex2Bytes(args[0])
	parsedVaa, err := vaa.Unmarshal(vaaData)
	if err != nil {
		cmd.PrintErrln("Failed to parse VAA:", err)
		os.Exit(1)
	}

	governanceAction, err := getGovernanceVaaAction(parsedVaa.Payload)
	if err != nil {
		cmd.PrintErrln("Failed to parse governance payload:", err)
		os.Exit(1)
	}

	var contractFunction func(opts *bind.TransactOpts, _vm []byte) (*types.Transaction, error)
	switch governanceAction {
	case 1:
		println("Governance Action: ContractUpgrade")
		contractFunction = t.SubmitContractUpgrade
	case 2:
		println("Governance Action: NewGuardianSet")
		contractFunction = t.SubmitNewGuardianSet
	case 3:
		println("Governance Action: SetMessageFee")
		contractFunction = t.SubmitSetMessageFee
	case 4:
		println("Governance Action: TransferFees")
		contractFunction = t.SubmitTransferFees
	default:
		cmd.PrintErrln("Unknow governance action")
		os.Exit(1)
	}

	res, err := contractFunction(&bind.TransactOpts{
		From:   addr,
		Signer: signer,
	}, vaaData)
	if err != nil {
		cmd.PrintErrln(err)
		os.Exit(1)
	}

	println("Posted tx. Hash:", res.Hash().String())
}

func getEthClient() (*ethclient.Client, error) {
	return ethclient.Dial(rpcUrl)
}

func getSigner(ethC *ethclient.Client) (func(address common.Address, transaction *types.Transaction) (*types.Transaction, error), common.Address, error) {
	cID, err := ethC.ChainID(context.Background())
	if err != nil {
		panic(err)
	}

	keyBytes := common.Hex2Bytes(key)
	key, err := crypto.ToECDSA(keyBytes)
	if err != nil {
		return nil, common.Address{}, err
	}

	return func(address common.Address, transaction *types.Transaction) (*types.Transaction, error) {
		return types.SignTx(transaction, types.NewEIP155Signer(cID), key)
	}, crypto.PubkeyToAddress(key.PublicKey), nil
}

func getTransactor(ethC *ethclient.Client) (*abi.AbiTransactor, error) {
	addr := common.HexToAddress(contractAddress)
	emptyAddr := common.Address{}
	if addr == emptyAddr {
		return nil, fmt.Errorf("invalid contract address")
	}

	t, err := abi.NewAbiTransactor(addr, ethC)
	if err != nil {
		return nil, err
	}

	return t, nil
}

func getGovernanceVaaAction(payload []byte) (uint8, error) {
	if len(payload) < 32+2+1 {
		return 0, fmt.Errorf("VAA payload does not contain a governance header")
	}

	return payload[32], nil
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
