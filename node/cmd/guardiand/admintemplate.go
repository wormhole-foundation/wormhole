package guardiand

import (
	"encoding/hex"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/btcsuite/btcutil/bech32"
	"github.com/ethereum/go-ethereum/common"
	"github.com/mr-tron/base58"
	"github.com/spf13/pflag"
	"github.com/tendermint/tendermint/libs/rand"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/encoding/prototext"

	"github.com/certusone/wormhole/node/pkg/devnet"
	nodev1 "github.com/certusone/wormhole/node/pkg/proto/node/v1"
)

var setUpdateNumGuardians *int
var templateGuardianIndex *int
var chainID *string
var address *string
var module *string

var circleIntegrationChainID *string
var circleIntegrationFinality *string
var circleIntegrationForeignEmitterChainID *string
var circleIntegrationForeignEmitterAddress *string
var circleIntegrationCircleDomain *string
var circleIntegrationNewImplementationAddress *string

var wormchainStoreCodeWasmHash *string

var wormchainInstantiateContractCodeId *string
var wormchainInstantiateContractInstantiationMsg *string
var wormchainInstantiateContractLabel *string

var wormchainMigrateContractCodeId *string
var wormchainMigrateContractContractAddress *string
var wormchainMigrateContractInstantiationMsg *string

var wormchainAllowlistInstantiateContractCodeId *string
var wormchainAllowlistInstantiateContractContractAddress *string

var ibcReceiverUpdateChannelChainTargetChainId *string
var ibcReceiverUpdateChannelChainChannelId *string
var ibcReceiverUpdateChannelChainChainId *string

func init() {
	governanceFlagSet := pflag.NewFlagSet("governance", pflag.ExitOnError)
	chainID = governanceFlagSet.String("chain-id", "", "Chain ID")
	address = governanceFlagSet.String("new-address", "", "New address (hex, base58 or bech32)")

	moduleFlagSet := pflag.NewFlagSet("module", pflag.ExitOnError)
	module = moduleFlagSet.String("module", "", "Module name")

	templateGuardianIndex = TemplateCmd.PersistentFlags().Int("idx", 3, "Default current guardian set index")

	setUpdateNumGuardians = AdminClientGuardianSetTemplateCmd.Flags().Int("num", 1, "Number of devnet guardians in example file")
	TemplateCmd.AddCommand(AdminClientGuardianSetTemplateCmd)

	AdminClientContractUpgradeTemplateCmd.Flags().AddFlagSet(governanceFlagSet)
	TemplateCmd.AddCommand(AdminClientContractUpgradeTemplateCmd)

	AdminClientTokenBridgeRegisterChainCmd.Flags().AddFlagSet(governanceFlagSet)
	AdminClientTokenBridgeRegisterChainCmd.Flags().AddFlagSet(moduleFlagSet)
	TemplateCmd.AddCommand(AdminClientTokenBridgeRegisterChainCmd)

	AdminClientTokenBridgeUpgradeContractCmd.Flags().AddFlagSet(governanceFlagSet)
	AdminClientTokenBridgeUpgradeContractCmd.Flags().AddFlagSet(moduleFlagSet)
	TemplateCmd.AddCommand(AdminClientTokenBridgeUpgradeContractCmd)

	AdminClientWormholeRelayerSetDefaultDeliveryProviderCmd.Flags().AddFlagSet(governanceFlagSet)
	TemplateCmd.AddCommand(AdminClientWormholeRelayerSetDefaultDeliveryProviderCmd)

	circleIntegrationChainIDFlagSet := pflag.NewFlagSet("circle-integ", pflag.ExitOnError)
	circleIntegrationChainID = circleIntegrationChainIDFlagSet.String("chain-id", "", "Target chain ID")

	circleIntegrationFinalityFlagSet := pflag.NewFlagSet("finality", pflag.ExitOnError)
	circleIntegrationFinality = circleIntegrationFinalityFlagSet.String("finality", "", "Desired wormhole finality")
	AdminClientCircleIntegrationUpdateWormholeFinalityCmd.Flags().AddFlagSet(circleIntegrationChainIDFlagSet)
	AdminClientCircleIntegrationUpdateWormholeFinalityCmd.Flags().AddFlagSet(circleIntegrationFinalityFlagSet)
	TemplateCmd.AddCommand(AdminClientCircleIntegrationUpdateWormholeFinalityCmd)

	circleIntegrationRegisterEmitterFlagSet := pflag.NewFlagSet("register", pflag.ExitOnError)
	circleIntegrationForeignEmitterChainID = circleIntegrationRegisterEmitterFlagSet.String("foreign-emitter-chain-id", "", "Foreign emitter chain ID")
	circleIntegrationForeignEmitterAddress = circleIntegrationRegisterEmitterFlagSet.String("foreign-emitter-address", "", "Foreign emitter address (hex, base58 or bech32)")
	circleIntegrationCircleDomain = circleIntegrationRegisterEmitterFlagSet.String("circle-domain", "", "Circle domain")
	AdminClientCircleIntegrationRegisterEmitterAndDomainCmd.Flags().AddFlagSet(circleIntegrationChainIDFlagSet)
	AdminClientCircleIntegrationRegisterEmitterAndDomainCmd.Flags().AddFlagSet(circleIntegrationRegisterEmitterFlagSet)
	TemplateCmd.AddCommand(AdminClientCircleIntegrationRegisterEmitterAndDomainCmd)

	circleIntegrationUpgradeContractImplementationFlagSet := pflag.NewFlagSet("upgrade", pflag.ExitOnError)
	circleIntegrationNewImplementationAddress = circleIntegrationUpgradeContractImplementationFlagSet.String("new-implementation-address", "", "New implementation address (hex, base58 or bech32)")
	AdminClientCircleIntegrationUpgradeContractImplementationCmd.Flags().AddFlagSet(circleIntegrationChainIDFlagSet)
	AdminClientCircleIntegrationUpgradeContractImplementationCmd.Flags().AddFlagSet(circleIntegrationUpgradeContractImplementationFlagSet)
	TemplateCmd.AddCommand(AdminClientCircleIntegrationUpgradeContractImplementationCmd)

	wormchainStoreCodeFlagSet := pflag.NewFlagSet("wormchain-store-code", pflag.ExitOnError)
	wormchainStoreCodeWasmHash = wormchainStoreCodeFlagSet.String("wasm-hash", "", "WASM Hash of the stored code")
	AdminClientWormchainStoreCodeCmd.Flags().AddFlagSet(wormchainStoreCodeFlagSet)
	TemplateCmd.AddCommand(AdminClientWormchainStoreCodeCmd)

	wormchainInstantiateContractFlagSet := pflag.NewFlagSet("wormchain-instantiate-contract", pflag.ExitOnError)
	wormchainInstantiateContractCodeId = wormchainInstantiateContractFlagSet.String("code-id", "", "code ID of the stored code")
	wormchainInstantiateContractLabel = wormchainInstantiateContractFlagSet.String("label", "", "label")
	wormchainInstantiateContractInstantiationMsg = wormchainInstantiateContractFlagSet.String("instantiation-msg", "", "instantiate message")
	AdminClientWormchainInstantiateContractCmd.Flags().AddFlagSet(wormchainInstantiateContractFlagSet)
	TemplateCmd.AddCommand(AdminClientWormchainInstantiateContractCmd)

	wormchainMigrateContractFlagSet := pflag.NewFlagSet("wormchain-migrate-contract", pflag.ExitOnError)
	wormchainMigrateContractCodeId = wormchainMigrateContractFlagSet.String("code-id", "", "code ID of the stored code")
	wormchainMigrateContractContractAddress = wormchainMigrateContractFlagSet.String("contract-address", "", "contract address")
	wormchainMigrateContractInstantiationMsg = wormchainMigrateContractFlagSet.String("instantiation-msg", "", "instantiate message")
	AdminClientWormchainMigrateContractCmd.Flags().AddFlagSet(wormchainMigrateContractFlagSet)
	TemplateCmd.AddCommand(AdminClientWormchainMigrateContractCmd)

	wormchainAllowlistInstantiateContractFlagSet := pflag.NewFlagSet("wormchain-instantiate-allowlist", pflag.ExitOnError)
	wormchainAllowlistInstantiateContractCodeId = wormchainAllowlistInstantiateContractFlagSet.String("code-id", "", "code ID of the stored code to allowlist wasm instantiate for")
	wormchainAllowlistInstantiateContractContractAddress = wormchainAllowlistInstantiateContractFlagSet.String("contract-address", "", "contract address to allowlist wasm instantiate for")
	AdminClientWormchainAllowlistInstantiateContractCmd.Flags().AddFlagSet(wormchainAllowlistInstantiateContractFlagSet)
	TemplateCmd.AddCommand(AdminClientWormchainAllowlistInstantiateContractCmd)

	// flags for the ibc-receiver-update-channel-chain command
	ibcReceiverUpdateChannelChainFlagSet := pflag.NewFlagSet("ibc-mapping", pflag.ExitOnError)
	ibcReceiverUpdateChannelChainTargetChainId = ibcReceiverUpdateChannelChainFlagSet.String("target-chain-id", "", "Target Chain ID for the governance VAA")
	ibcReceiverUpdateChannelChainChannelId = ibcReceiverUpdateChannelChainFlagSet.String("channel-id", "", "IBC Channel ID on Wormchain")
	ibcReceiverUpdateChannelChainChainId = ibcReceiverUpdateChannelChainFlagSet.String("chain-id", "", "IBC Chain ID that the channel ID corresponds to")
	AdminClientIbcReceiverUpdateChannelChainCmd.Flags().AddFlagSet(ibcReceiverUpdateChannelChainFlagSet)
	TemplateCmd.AddCommand(AdminClientIbcReceiverUpdateChannelChainCmd)
}

var TemplateCmd = &cobra.Command{
	Use:   "template",
	Short: "Guardian governance VAA template commands ",
}

var AdminClientGuardianSetTemplateCmd = &cobra.Command{
	Use:   "guardian-set-update",
	Short: "Generate an empty guardian set template",
	Run:   runGuardianSetTemplate,
}

var AdminClientContractUpgradeTemplateCmd = &cobra.Command{
	Use:   "contract-upgrade",
	Short: "Generate an empty contract upgrade template",
	Run:   runContractUpgradeTemplate,
}

var AdminClientTokenBridgeRegisterChainCmd = &cobra.Command{
	Use:   "token-bridge-register-chain",
	Short: "Generate an empty token bridge chain registration template at specified path",
	Run:   runTokenBridgeRegisterChainTemplate,
}

var AdminClientTokenBridgeUpgradeContractCmd = &cobra.Command{
	Use:   "token-bridge-upgrade-contract",
	Short: "Generate an empty token bridge contract upgrade template at specified path",
	Run:   runTokenBridgeUpgradeContractTemplate,
}

var AdminClientCircleIntegrationUpdateWormholeFinalityCmd = &cobra.Command{
	Use:   "circle-integration-update-wormhole-finality",
	Short: "Generate an empty circle integration update wormhole finality template at specified path",
	Run:   runCircleIntegrationUpdateWormholeFinalityTemplate,
}

var AdminClientCircleIntegrationRegisterEmitterAndDomainCmd = &cobra.Command{
	Use:   "circle-integration-register-emitter-and-domain",
	Short: "Generate an empty circle integration register emitter and domain template at specified path",
	Run:   runCircleIntegrationRegisterEmitterAndDomainTemplate,
}

var AdminClientCircleIntegrationUpgradeContractImplementationCmd = &cobra.Command{
	Use:   "circle-integration-upgrade-contract-implementation",
	Short: "Generate an empty circle integration upgrade contract implementation template at specified path",
	Run:   runCircleIntegrationUpgradeContractImplementationTemplate,
}

var AdminClientWormchainStoreCodeCmd = &cobra.Command{
	Use:   "wormchain-store-code",
	Short: "Generate an empty wormchain store code template at specified path",
	Run:   runWormchainStoreCodeTemplate,
}

var AdminClientWormchainInstantiateContractCmd = &cobra.Command{
	Use:   "wormchain-instantiate-contract",
	Short: "Generate an empty wormchain instantiate contract template at specified path",
	Run:   runWormchainInstantiateContractTemplate,
}

var AdminClientWormchainMigrateContractCmd = &cobra.Command{
	Use:   "wormchain-migrate-contract",
	Short: "Generate an empty wormchain migrate contract template at specified path",
	Run:   runWormchainMigrateContractTemplate,
}

var AdminClientWormchainAllowlistInstantiateContractCmd = &cobra.Command{
	Use:   "wormchain-allowlist-instantiate-contract",
	Short: "Generate an empty wormchain allowlist instantiate contract template at specified path",
	Run:   runWormchainAllowlistInstantiateContractTemplate,
}

var AdminClientIbcReceiverUpdateChannelChainCmd = &cobra.Command{
	Use:   "ibc-receiver-update-channel-chain",
	Short: "Generate an empty ibc receiver channelId to chainId mapping update template at specified path",
	Run:   runIbcReceiverUpdateChannelChainTemplate,
}

var AdminClientWormholeRelayerSetDefaultDeliveryProviderCmd = &cobra.Command{
	Use:   "wormhole-relayer-set-default-delivery-provider",
	Short: "Generate a 'set default delivery provider' template for specified chain and address",
	Run:   runWormholeRelayerSetDefaultDeliveryProviderTemplate,
}

func runGuardianSetTemplate(cmd *cobra.Command, args []string) {
	// Use deterministic devnet addresses as examples in the template, such that this doubles as a test fixture.
	guardians := make([]*nodev1.GuardianSetUpdate_Guardian, *setUpdateNumGuardians)
	for i := 0; i < *setUpdateNumGuardians; i++ {
		k := devnet.InsecureDeterministicEcdsaKeyByIndex(crypto.S256(), uint64(i))
		guardians[i] = &nodev1.GuardianSetUpdate_Guardian{
			Pubkey: crypto.PubkeyToAddress(k.PublicKey).Hex(),
			Name:   fmt.Sprintf("Example validator %d", i),
		}
	}

	m := &nodev1.InjectGovernanceVAARequest{
		CurrentSetIndex: uint32(*templateGuardianIndex),
		Messages: []*nodev1.GovernanceMessage{
			{
				Sequence: rand.Uint64(),
				Nonce:    rand.Uint32(),
				Payload: &nodev1.GovernanceMessage_GuardianSet{
					GuardianSet: &nodev1.GuardianSetUpdate{Guardians: guardians},
				},
			},
		},
	}

	b, err := prototext.MarshalOptions{Multiline: true}.Marshal(m)
	if err != nil {
		panic(err)
	}
	fmt.Print(string(b))
}

func runContractUpgradeTemplate(cmd *cobra.Command, args []string) {
	address, err := parseAddress(*address)
	if err != nil {
		log.Fatal(err)
	}
	chainID, err := parseChainID(*chainID)
	if err != nil {
		log.Fatal(err)
	}

	m := &nodev1.InjectGovernanceVAARequest{
		CurrentSetIndex: uint32(*templateGuardianIndex),
		Messages: []*nodev1.GovernanceMessage{
			{
				Sequence: rand.Uint64(),
				Nonce:    rand.Uint32(),
				Payload: &nodev1.GovernanceMessage_ContractUpgrade{
					ContractUpgrade: &nodev1.ContractUpgrade{
						ChainId:     uint32(chainID),
						NewContract: address,
					},
				},
			},
		},
	}

	b, err := prototext.MarshalOptions{Multiline: true}.Marshal(m)
	if err != nil {
		panic(err)
	}
	fmt.Print(string(b))
}
func runTokenBridgeRegisterChainTemplate(cmd *cobra.Command, args []string) {
	address, err := parseAddress(*address)
	if err != nil {
		log.Fatal(err)
	}
	chainID, err := parseChainID(*chainID)
	if err != nil {
		log.Fatal(err)
	}

	m := &nodev1.InjectGovernanceVAARequest{
		CurrentSetIndex: uint32(*templateGuardianIndex),
		Messages: []*nodev1.GovernanceMessage{
			{
				Sequence: rand.Uint64(),
				Nonce:    rand.Uint32(),
				Payload: &nodev1.GovernanceMessage_BridgeRegisterChain{
					BridgeRegisterChain: &nodev1.BridgeRegisterChain{
						Module:         *module,
						ChainId:        uint32(chainID),
						EmitterAddress: address,
					},
				},
			},
		},
	}

	b, err := prototext.MarshalOptions{Multiline: true}.Marshal(m)
	if err != nil {
		panic(err)
	}
	fmt.Print(string(b))
}

func runTokenBridgeUpgradeContractTemplate(cmd *cobra.Command, args []string) {
	address, err := parseAddress(*address)
	if err != nil {
		log.Fatal(err)
	}
	chainID, err := parseChainID(*chainID)
	if err != nil {
		log.Fatal(err)
	}

	m := &nodev1.InjectGovernanceVAARequest{
		CurrentSetIndex: uint32(*templateGuardianIndex),
		Messages: []*nodev1.GovernanceMessage{
			{
				Sequence: rand.Uint64(),
				Nonce:    rand.Uint32(),
				Payload: &nodev1.GovernanceMessage_BridgeContractUpgrade{
					BridgeContractUpgrade: &nodev1.BridgeUpgradeContract{
						Module:        *module,
						TargetChainId: uint32(chainID),
						NewContract:   address,
					},
				},
			},
		},
	}

	b, err := prototext.MarshalOptions{Multiline: true}.Marshal(m)
	if err != nil {
		panic(err)
	}
	fmt.Print(string(b))
}

func runCircleIntegrationUpdateWormholeFinalityTemplate(cmd *cobra.Command, args []string) {
	if *circleIntegrationChainID == "" {
		log.Fatal("--chain-id must be specified.")
	}
	chainID, err := parseChainID(*circleIntegrationChainID)
	if err != nil {
		log.Fatal("failed to parse chain id:", err)
	}
	if *circleIntegrationFinality == "" {
		log.Fatal("--finality must be specified.")
	}
	finality, err := strconv.ParseUint(*circleIntegrationFinality, 10, 8)
	if err != nil {
		log.Fatal("failed to parse finality as uint8: ", err)
	}

	m := &nodev1.InjectGovernanceVAARequest{
		CurrentSetIndex: uint32(*templateGuardianIndex),
		Messages: []*nodev1.GovernanceMessage{
			{
				Sequence: rand.Uint64(),
				Nonce:    rand.Uint32(),
				Payload: &nodev1.GovernanceMessage_CircleIntegrationUpdateWormholeFinality{
					CircleIntegrationUpdateWormholeFinality: &nodev1.CircleIntegrationUpdateWormholeFinality{
						TargetChainId: uint32(chainID),
						Finality:      uint32(finality),
					},
				},
			},
		},
	}

	b, err := prototext.MarshalOptions{Multiline: true}.Marshal(m)
	if err != nil {
		panic(err)
	}
	fmt.Print(string(b))
}

func runCircleIntegrationRegisterEmitterAndDomainTemplate(cmd *cobra.Command, args []string) {
	if *circleIntegrationChainID == "" {
		log.Fatal("--chain-id must be specified.")
	}
	chainID, err := parseChainID(*circleIntegrationChainID)
	if err != nil {
		log.Fatal("failed to parse chain id:", err)
	}
	if *circleIntegrationForeignEmitterChainID == "" {
		log.Fatal("--foreign-emitter-chain-id must be specified.")
	}
	foreignEmitterChainId, err := parseChainID(*circleIntegrationForeignEmitterChainID)
	if err != nil {
		log.Fatal("failed to parse foreign emitter chain id as uint8:", err)
	}
	if *circleIntegrationForeignEmitterAddress == "" {
		log.Fatal("--foreign-emitter-address must be specified.")
	}
	foreignEmitterAddress, err := parseAddress(*circleIntegrationForeignEmitterAddress)
	if err != nil {
		log.Fatal("failed to parse foreign emitter address: ", err)
	}
	if *circleIntegrationCircleDomain == "" {
		log.Fatal("--circle-domain must be specified.")
	}
	circleDomain, err := strconv.ParseUint(*circleIntegrationCircleDomain, 10, 32)
	if err != nil {
		log.Fatal("failed to parse circle domain as uint32:", err)
	}

	m := &nodev1.InjectGovernanceVAARequest{
		CurrentSetIndex: uint32(*templateGuardianIndex),
		Messages: []*nodev1.GovernanceMessage{
			{
				Sequence: rand.Uint64(),
				Nonce:    rand.Uint32(),
				Payload: &nodev1.GovernanceMessage_CircleIntegrationRegisterEmitterAndDomain{
					CircleIntegrationRegisterEmitterAndDomain: &nodev1.CircleIntegrationRegisterEmitterAndDomain{
						TargetChainId:         uint32(chainID),
						ForeignEmitterChainId: uint32(foreignEmitterChainId),
						ForeignEmitterAddress: foreignEmitterAddress,
						CircleDomain:          uint32(circleDomain),
					},
				},
			},
		},
	}

	b, err := prototext.MarshalOptions{Multiline: true}.Marshal(m)
	if err != nil {
		panic(err)
	}
	fmt.Print(string(b))
}

func runCircleIntegrationUpgradeContractImplementationTemplate(cmd *cobra.Command, args []string) {
	if *circleIntegrationChainID == "" {
		log.Fatal("--chain-id must be specified.")
	}
	chainID, err := parseChainID(*circleIntegrationChainID)
	if err != nil {
		log.Fatal("failed to parse chain id:", err)
	}
	if *circleIntegrationNewImplementationAddress == "" {
		log.Fatal("--new-implementation-address must be specified.")
	}
	newImplementationAddress, err := parseAddress(*circleIntegrationNewImplementationAddress)
	if err != nil {
		log.Fatal("failed to parse new implementation address: ", err)
	}

	m := &nodev1.InjectGovernanceVAARequest{
		CurrentSetIndex: uint32(*templateGuardianIndex),
		Messages: []*nodev1.GovernanceMessage{
			{
				Sequence: rand.Uint64(),
				Nonce:    rand.Uint32(),
				Payload: &nodev1.GovernanceMessage_CircleIntegrationUpgradeContractImplementation{
					CircleIntegrationUpgradeContractImplementation: &nodev1.CircleIntegrationUpgradeContractImplementation{
						TargetChainId:            uint32(chainID),
						NewImplementationAddress: newImplementationAddress,
					},
				},
			},
		},
	}

	b, err := prototext.MarshalOptions{Multiline: true}.Marshal(m)
	if err != nil {
		panic(err)
	}
	fmt.Print(string(b))
}

func runWormchainStoreCodeTemplate(cmd *cobra.Command, args []string) {
	if *wormchainStoreCodeWasmHash == "" {
		log.Fatal("--wasm-hash must be specified.")
	}

	// Validate the string is valid hex.
	buf, err := hex.DecodeString(*wormchainStoreCodeWasmHash)
	if err != nil {
		log.Fatal("invalid wasm-hash (expected hex): %w", err)
	}

	// Validate the string is the correct length.
	if len(buf) != 32 {
		log.Fatalf("wasm-hash (expected 32 bytes but received %d bytes)", len(buf))
	}

	m := &nodev1.InjectGovernanceVAARequest{
		CurrentSetIndex: uint32(*templateGuardianIndex),
		Messages: []*nodev1.GovernanceMessage{
			{
				Sequence: rand.Uint64(),
				Nonce:    rand.Uint32(),
				Payload: &nodev1.GovernanceMessage_WormchainStoreCode{
					WormchainStoreCode: &nodev1.WormchainStoreCode{
						WasmHash: string(*wormchainStoreCodeWasmHash),
					},
				},
			},
		},
	}

	b, err := prototext.MarshalOptions{Multiline: true}.Marshal(m)
	if err != nil {
		panic(err)
	}
	fmt.Print(string(b))
}

func runWormchainInstantiateContractTemplate(cmd *cobra.Command, args []string) {
	if *wormchainInstantiateContractCodeId == "" {
		log.Fatal("--code-id must be specified.")
	}
	codeId, err := strconv.ParseUint(*wormchainInstantiateContractCodeId, 10, 64)
	if err != nil {
		log.Fatal("failed to parse code-id as uint64: ", err)
	}
	if *wormchainInstantiateContractLabel == "" {
		log.Fatal("--label must be specified.")
	}
	if *wormchainInstantiateContractInstantiationMsg == "" {
		log.Fatal("--instantiation-msg must be specified.")
	}

	m := &nodev1.InjectGovernanceVAARequest{
		CurrentSetIndex: uint32(*templateGuardianIndex),
		Messages: []*nodev1.GovernanceMessage{
			{
				Sequence: rand.Uint64(),
				Nonce:    rand.Uint32(),
				Payload: &nodev1.GovernanceMessage_WormchainInstantiateContract{
					WormchainInstantiateContract: &nodev1.WormchainInstantiateContract{
						CodeId:           codeId,
						Label:            *wormchainInstantiateContractLabel,
						InstantiationMsg: *wormchainInstantiateContractInstantiationMsg,
					},
				},
			},
		},
	}

	b, err := prototext.MarshalOptions{Multiline: true}.Marshal(m)
	if err != nil {
		panic(err)
	}
	fmt.Print(string(b))
}

func runWormchainMigrateContractTemplate(cmd *cobra.Command, args []string) {
	if *wormchainMigrateContractCodeId == "" {
		log.Fatal("--code-id must be specified.")
	}
	codeId, err := strconv.ParseUint(*wormchainMigrateContractCodeId, 10, 64)
	if err != nil {
		log.Fatal("failed to parse code-id as uint64: ", err)
	}
	if *wormchainMigrateContractContractAddress == "" {
		log.Fatal("--contract-address must be specified.")
	}
	if *wormchainMigrateContractInstantiationMsg == "" {
		log.Fatal("--instantiate-msg must be specified.")
	}

	m := &nodev1.InjectGovernanceVAARequest{
		CurrentSetIndex: uint32(*templateGuardianIndex),
		Messages: []*nodev1.GovernanceMessage{
			{
				Sequence: rand.Uint64(),
				Nonce:    rand.Uint32(),
				Payload: &nodev1.GovernanceMessage_WormchainMigrateContract{
					WormchainMigrateContract: &nodev1.WormchainMigrateContract{
						CodeId:           codeId,
						Contract:         *wormchainMigrateContractContractAddress,
						InstantiationMsg: *wormchainMigrateContractInstantiationMsg,
					},
				},
			},
		},
	}

	b, err := prototext.MarshalOptions{Multiline: true}.Marshal(m)
	if err != nil {
		panic(err)
	}
	fmt.Print(string(b))
}

func runWormchainAllowlistInstantiateContractTemplate(cmd *cobra.Command, args []string) {
	if *wormchainAllowlistInstantiateContractCodeId == "" {
		log.Fatal("--code-id must be specified")
	}
	codeId, err := strconv.ParseUint(*wormchainAllowlistInstantiateContractCodeId, 10, 64)
	if err != nil {
		log.Fatal("failed to parse code-id as utin64: ", err)
	}
	if *wormchainAllowlistInstantiateContractContractAddress == "" {
		log.Fatal("--contract-address must be specified")
	}

	m := &nodev1.InjectGovernanceVAARequest{
		CurrentSetIndex: uint32(*templateGuardianIndex),
		Messages: []*nodev1.GovernanceMessage{
			{
				Sequence: rand.Uint64(),
				Nonce:    rand.Uint32(),
				Payload: &nodev1.GovernanceMessage_WormchainAllowlistInstantiateContract{
					WormchainAllowlistInstantiateContract: &nodev1.WormchainAllowlistInstantiateContract{
						CodeId:   codeId,
						Contract: *wormchainAllowlistInstantiateContractContractAddress,
					},
				},
			},
		},
	}

	b, err := prototext.MarshalOptions{Multiline: true}.Marshal(m)
	if err != nil {
		panic(err)
	}
	fmt.Print(string(b))
}

func runIbcReceiverUpdateChannelChainTemplate(cmd *cobra.Command, args []string) {
	if *ibcReceiverUpdateChannelChainTargetChainId == "" {
		log.Fatal("--target-chain-id must be specified")
	}
	targetChainId, err := parseChainID(*ibcReceiverUpdateChannelChainTargetChainId)
	if err != nil {
		log.Fatal("failed to parse chain id: ", err)
	}

	if *ibcReceiverUpdateChannelChainChannelId == "" {
		log.Fatal("--channel-id must be specified")
	}
	if len(*ibcReceiverUpdateChannelChainChannelId) > 64 {
		log.Fatal("invalid channel id length, must be <= 64")
	}

	if *ibcReceiverUpdateChannelChainChainId == "" {
		log.Fatal("--chain-id must be specified")
	}
	chainId, err := parseChainID(*ibcReceiverUpdateChannelChainChainId)
	if err != nil {
		log.Fatal("failed to parse chain id: ", err)
	}

	m := &nodev1.InjectGovernanceVAARequest{
		CurrentSetIndex: uint32(*templateGuardianIndex),
		Messages: []*nodev1.GovernanceMessage{
			{
				Sequence: rand.Uint64(),
				Nonce:    rand.Uint32(),
				Payload: &nodev1.GovernanceMessage_IbcReceiverUpdateChannelChain{
					IbcReceiverUpdateChannelChain: &nodev1.IbcReceiverUpdateChannelChain{
						TargetChainId: uint32(targetChainId),
						ChannelId:     *ibcReceiverUpdateChannelChainChannelId,
						ChainId:       uint32(chainId),
					},
				},
			},
		},
	}

	b, err := prototext.MarshalOptions{Multiline: true}.Marshal(m)
	if err != nil {
		panic(err)
	}
	fmt.Print(string(b))

}

func runWormholeRelayerSetDefaultDeliveryProviderTemplate(cmd *cobra.Command, args []string) {
	address, err := parseAddress(*address)
	if err != nil {
		log.Fatal(err)
	}
	chainID, err := parseChainID(*chainID)
	if err != nil {
		log.Fatal(err)
	}

	m := &nodev1.InjectGovernanceVAARequest{
		CurrentSetIndex: uint32(*templateGuardianIndex),
		Messages: []*nodev1.GovernanceMessage{
			{
				Sequence: rand.Uint64(),
				Nonce:    rand.Uint32(),
				Payload: &nodev1.GovernanceMessage_WormholeRelayerSetDefaultDeliveryProvider{
					WormholeRelayerSetDefaultDeliveryProvider: &nodev1.WormholeRelayerSetDefaultDeliveryProvider{
						ChainId:                           uint32(chainID),
						NewDefaultDeliveryProviderAddress: address,
					},
				},
			},
		},
	}

	b, err := prototext.MarshalOptions{Multiline: true}.Marshal(m)
	if err != nil {
		panic(err)
	}
	fmt.Print(string(b))
}

// parseAddress parses either a hex-encoded address and returns
// a left-padded 32 byte hex string.
func parseAddress(s string) (string, error) {
	// try base58
	b, err := base58.Decode(s)
	if err == nil {
		return leftPadAddress(b)
	}

	// try bech32
	_, b, err = bech32.Decode(s)
	if err == nil {
		return leftPadAddress(b)
	}

	// try hex
	if len(s) > 2 && strings.ToLower(s[:2]) == "0x" {
		s = s[2:]
	}

	a, err := hex.DecodeString(s)
	if err != nil {
		return "", fmt.Errorf("invalid hex address: %v", err)
	}
	return leftPadAddress(a)
}

func leftPadAddress(a []byte) (string, error) {
	if len(a) > 32 {
		return "", fmt.Errorf("address longer than 32 bytes")
	}
	return hex.EncodeToString(common.LeftPadBytes(a, 32)), nil
}

// parseChainID parses a human-readable chain name or a chain ID.
func parseChainID(name string) (vaa.ChainID, error) {
	s, err := vaa.ChainIDFromString(name)
	if err == nil {
		return s, nil
	}

	// parse as uint32
	i, err := strconv.ParseUint(name, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("failed to parse as name or uint32: %v", err)
	}

	return vaa.ChainID(i), nil
}
