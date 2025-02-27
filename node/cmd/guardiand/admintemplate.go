package guardiand

import (
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"math/big"
	"strconv"
	"strings"

	"github.com/btcsuite/btcutil/bech32"
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
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

var wormchainWasmInstantiateAllowlistCodeId *string
var wormchainWasmInstantiateAllowlistContractAddress *string

var gatewayScheduleUpgradeName *string
var gatewayScheduleUpgradeHeight *string
var gatewayIbcComposabilityMwContractAddress *string

var ibcUpdateChannelChainTargetChainId *string
var ibcUpdateChannelChainChannelId *string
var ibcUpdateChannelChainChainId *string

var recoverChainIdEvmChainId *string
var recoverChainIdNewChainId *string

var accountantModifyBalanceModule *string
var accountantModifyBalanceTargetChainId *string
var accountantModifyBalanceSequence *string
var accountantModifyBalanceChainId *string
var accountantModifyBalanceTokenChainId *string
var accountantModifyBalanceTokenAddress *string
var accountantModifyBalanceAction *string
var accountantModifyBalanceAmount *string
var accountantModifyBalanceReason *string

var governanceContractAddress *string
var governanceTargetAddress *string
var governanceTargetChain *string
var governanceCallData *string

func init() {
	governanceFlagSet := pflag.NewFlagSet("governance", pflag.ExitOnError)
	chainID = governanceFlagSet.String("chain-id", "", "Chain ID")
	address = governanceFlagSet.String("new-address", "", "New address (hex, base58 or bech32)")

	moduleFlagSet := pflag.NewFlagSet("module", pflag.ExitOnError)
	module = moduleFlagSet.String("module", "", "Module name")

	templateGuardianIndex = TemplateCmd.PersistentFlags().Int("idx", 4, "Default current guardian set index")

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

	// flags for the wormchain add/delete wasm instantiate allowlist commands
	wormchainWasmInstantiateAllowlistFlagSet := pflag.NewFlagSet("wormchain-wasm-instantiate-allowlist", pflag.ExitOnError)
	wormchainWasmInstantiateAllowlistCodeId = wormchainWasmInstantiateAllowlistFlagSet.String("code-id", "", "code ID of the stored code to add/delete allowlist wasm instantiate for")
	wormchainWasmInstantiateAllowlistContractAddress = wormchainWasmInstantiateAllowlistFlagSet.String("contract-address", "", "contract address to add/delete allowlist wasm instantiate for")
	AdminClientWormchainAddWasmInstantiateAllowlistCmd.Flags().AddFlagSet(wormchainWasmInstantiateAllowlistFlagSet)
	AdminClientWormchainDeleteWasmInstantiateAllowlistCmd.Flags().AddFlagSet(wormchainWasmInstantiateAllowlistFlagSet)
	TemplateCmd.AddCommand(AdminClientWormchainAddWasmInstantiateAllowlistCmd)
	TemplateCmd.AddCommand(AdminClientWormchainDeleteWasmInstantiateAllowlistCmd)

	// flags for the gateway-ibc-composability-mw-set-contract command
	gatewayIbcComposabilityMwFlagSet := pflag.NewFlagSet("gateway-ibc-composability-mw-set-contract", pflag.ExitOnError)
	gatewayIbcComposabilityMwContractAddress = gatewayIbcComposabilityMwFlagSet.String("contract-address", "", "contract address to set in the ibc composability middleware")
	AdminClientGatewayIbcComposabilityMwSetContractCmd.Flags().AddFlagSet(gatewayIbcComposabilityMwFlagSet)
	TemplateCmd.AddCommand(AdminClientGatewayIbcComposabilityMwSetContractCmd)

	// flags for the gateway-schedule-upgrade command
	gatewayScheduleUpgradeFlagSet := pflag.NewFlagSet("gateway-schedule-upgrade", pflag.ExitOnError)
	gatewayScheduleUpgradeName = gatewayScheduleUpgradeFlagSet.String("name", "", "Scheduled upgrade name")
	gatewayScheduleUpgradeHeight = gatewayScheduleUpgradeFlagSet.String("height", "", "Scheduled upgrade height")
	AdminClientGatewayScheduleUpgradeCmd.Flags().AddFlagSet(gatewayScheduleUpgradeFlagSet)
	TemplateCmd.AddCommand(AdminClientGatewayScheduleUpgradeCmd)

	// AdminClientGatewayCancelUpgradeCmd doesn't have any flags
	TemplateCmd.AddCommand(AdminClientGatewayCancelUpgradeCmd)

	// flags for the ibc-receiver-update-channel-chain and ibc-translator-update-channel-chain commands
	ibcUpdateChannelChainFlagSet := pflag.NewFlagSet("ibc-mapping", pflag.ExitOnError)
	ibcUpdateChannelChainTargetChainId = ibcUpdateChannelChainFlagSet.String("target-chain-id", "", "Target Chain ID for the governance VAA")
	ibcUpdateChannelChainChannelId = ibcUpdateChannelChainFlagSet.String("channel-id", "", "IBC Channel ID on Wormchain")
	ibcUpdateChannelChainChainId = ibcUpdateChannelChainFlagSet.String("chain-id", "", "IBC Chain ID that the channel ID corresponds to")
	AdminClientIbcReceiverUpdateChannelChainCmd.Flags().AddFlagSet(ibcUpdateChannelChainFlagSet)
	AdminClientIbcTranslatorUpdateChannelChainCmd.Flags().AddFlagSet(ibcUpdateChannelChainFlagSet)
	TemplateCmd.AddCommand(AdminClientIbcReceiverUpdateChannelChainCmd)
	TemplateCmd.AddCommand(AdminClientIbcTranslatorUpdateChannelChainCmd)

	// flags for the recover-chain-id command
	recoverChainIdFlagSet := pflag.NewFlagSet("recover-chain-id", pflag.ExitOnError)
	recoverChainIdEvmChainId = recoverChainIdFlagSet.String("evm-chain-id", "", "EVM Chain ID to recover")
	recoverChainIdNewChainId = recoverChainIdFlagSet.String("new-chain-id", "", "New Chain ID to recover to")
	AdminClientRecoverChainIdCmd.Flags().AddFlagSet(recoverChainIdFlagSet)
	AdminClientRecoverChainIdCmd.Flags().AddFlagSet(moduleFlagSet)
	TemplateCmd.AddCommand(AdminClientRecoverChainIdCmd)

	// flags for the accountant-modify-balance command
	accountantModifyBalanceFlagSet := pflag.NewFlagSet("accountant-modify-balance", pflag.ExitOnError)
	accountantModifyBalanceModule = accountantModifyBalanceFlagSet.String("module", "GlobalAccountant", "Module identifier of the accountant")
	accountantModifyBalanceTargetChainId = accountantModifyBalanceFlagSet.String("target-chain-id", "", "ID of the chain to receive this modification")
	accountantModifyBalanceSequence = accountantModifyBalanceFlagSet.String("sequence", "", "The sequence number of this modification.  Each modification must be uniquely identifiable just by its sequence number")
	accountantModifyBalanceChainId = accountantModifyBalanceFlagSet.String("chain-id", "", "Chain ID of the account to be modified")
	accountantModifyBalanceTokenChainId = accountantModifyBalanceFlagSet.String("token-chain-id", "", "Chain ID of the native chain for the token")
	accountantModifyBalanceTokenAddress = accountantModifyBalanceFlagSet.String("token-address", "", "Address of the token on its native chain, hex string encoded")
	accountantModifyBalanceAction = accountantModifyBalanceFlagSet.String("action", "", "Kind of modification to be made (1 = add, 2 = sub)")
	accountantModifyBalanceAmount = accountantModifyBalanceFlagSet.String("amount", "", `Amount to be modified (decimal formatted string indicating the"raw" amount, not adjusted by the decimals of the token`)
	accountantModifyBalanceReason = accountantModifyBalanceFlagSet.String("reason", "", "human-readable reason for the modification")
	AdminClientAccountantModifyBalanceCmd.Flags().AddFlagSet(accountantModifyBalanceFlagSet)
	AdminClientAccountantModifyBalanceCmd.Flags().AddFlagSet(moduleFlagSet)
	TemplateCmd.AddCommand(AdminClientAccountantModifyBalanceCmd)

	// flags for general-purpose governance call command
	generalPurposeGovernanceFlagSet := pflag.NewFlagSet("general-purpose-governance", pflag.ExitOnError)
	governanceContractAddress = generalPurposeGovernanceFlagSet.String("governance-contract", "", "Governance contract address")
	governanceTargetAddress = generalPurposeGovernanceFlagSet.String("target-address", "", "Address of the governed contract")
	governanceCallData = generalPurposeGovernanceFlagSet.String("call-data", "", "calldata")
	governanceTargetChain = generalPurposeGovernanceFlagSet.String("chain-id", "", "Chain ID")
	// evm call command
	AdminClientGeneralPurposeGovernanceEvmCallCmd.Flags().AddFlagSet(generalPurposeGovernanceFlagSet)
	TemplateCmd.AddCommand(AdminClientGeneralPurposeGovernanceEvmCallCmd)
	// solana call command
	AdminClientGeneralPurposeGovernanceSolanaCallCmd.Flags().AddFlagSet(generalPurposeGovernanceFlagSet)
	TemplateCmd.AddCommand(AdminClientGeneralPurposeGovernanceSolanaCallCmd)
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

var AdminClientRecoverChainIdCmd = &cobra.Command{
	Use:   "recover-chain-id",
	Short: "Generate an empty recover chain id template at specified path",
	Run:   runRecoverChainIdTemplate,
}

var AdminClientAccountantModifyBalanceCmd = &cobra.Command{
	Use:   "accountant-modify-balance",
	Short: "Generate an empty accountant modify balance template at specified path",
	Run:   runAccountantModifyBalanceTemplate,
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

var AdminClientWormchainAddWasmInstantiateAllowlistCmd = &cobra.Command{
	Use:   "wormchain-add-wasm-instantiate-allowlist",
	Short: "Generate an empty wormchain add wasm instantiate allowlist template at specified path",
	Run:   runWormchainAddWasmInstantiateAllowlistTemplate,
}

var AdminClientWormchainDeleteWasmInstantiateAllowlistCmd = &cobra.Command{
	Use:   "wormchain-delete-wasm-instantiate-allowlist",
	Short: "Generate an empty wormchain delete wasm instantiate allowlist template at specified path",
	Run:   runWormchainDeleteWasmInstantiateAllowlistTemplate,
}

var AdminClientGatewayScheduleUpgradeCmd = &cobra.Command{
	Use:   "gateway-schedule-upgrade",
	Short: "Schedule an upgrade on Gateway with a specified name for a specified height",
	Run:   runGatewayScheduleUpgradeTemplate,
}

var AdminClientGatewayCancelUpgradeCmd = &cobra.Command{
	Use:   "gateway-cancel-upgrade",
	Short: "Cancel a scheduled upgrade on Gateway",
	Run:   runGatewayCancelUpgradeTemplate,
}

var AdminClientGatewayIbcComposabilityMwSetContractCmd = &cobra.Command{
	Use:   "gateway-ibc-composability-mw-set-contract",
	Short: "Set the contract that the IBC Composability middleware will query",
	Run:   runGatewayIbcComposabilityMwSetContractTemplate,
}

var AdminClientIbcReceiverUpdateChannelChainCmd = &cobra.Command{
	Use:   "ibc-receiver-update-channel-chain",
	Short: "Generate an empty ibc receiver channelId to chainId mapping update template at specified path",
	Run:   runIbcReceiverUpdateChannelChainTemplate,
}

var AdminClientIbcTranslatorUpdateChannelChainCmd = &cobra.Command{
	Use:   "ibc-translator-update-channel-chain",
	Short: "Generate an empty ibc translator channelId to chainId mapping update template at specified path",
	Run:   runIbcTranslatorUpdateChannelChainTemplate,
}

var AdminClientWormholeRelayerSetDefaultDeliveryProviderCmd = &cobra.Command{
	Use:   "wormhole-relayer-set-default-delivery-provider",
	Short: "Generate a 'set default delivery provider' template for specified chain and address",
	Run:   runWormholeRelayerSetDefaultDeliveryProviderTemplate,
}

var AdminClientGeneralPurposeGovernanceEvmCallCmd = &cobra.Command{
	Use:   "governance-evm-call",
	Short: "Generate a 'general purpose evm governance call' template for specified chain and address",
	Run:   runGeneralPurposeGovernanceEvmCallTemplate,
}

var AdminClientGeneralPurposeGovernanceSolanaCallCmd = &cobra.Command{
	Use:   "governance-solana-call",
	Short: "Generate a 'general purpose solana governance call' template for specified chain and address",
	Run:   runGeneralPurposeGovernanceSolanaCallTemplate,
}

func runGuardianSetTemplate(cmd *cobra.Command, args []string) {
	// Use deterministic devnet addresses as examples in the template, such that this doubles as a test fixture.
	guardians := make([]*nodev1.GuardianSetUpdate_Guardian, *setUpdateNumGuardians)
	for i := 0; i < *setUpdateNumGuardians; i++ {
		k := devnet.InsecureDeterministicEcdsaKeyByIndex(crypto.S256(), uint64(i)) // #nosec G115 -- Number of guardians will never overflow here
		guardians[i] = &nodev1.GuardianSetUpdate_Guardian{
			Pubkey: crypto.PubkeyToAddress(k.PublicKey).Hex(),
			Name:   fmt.Sprintf("Example validator %d", i),
		}
	}

	m := &nodev1.InjectGovernanceVAARequest{
		CurrentSetIndex: uint32(*templateGuardianIndex), // #nosec G115 -- Number of guardians will never overflow here
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
		log.Fatal("failed to marshal request: ", err)
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
		CurrentSetIndex: uint32(*templateGuardianIndex), // #nosec G115 -- This will never overflow
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
		log.Fatal("failed to marshal request: ", err)
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
		CurrentSetIndex: uint32(*templateGuardianIndex), // #nosec G115 -- This will never overflow
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
		log.Fatal("failed to marshal request: ", err)
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
		CurrentSetIndex: uint32(*templateGuardianIndex), // #nosec G115 -- This will never overflow
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
		log.Fatal("failed to marshal request: ", err)
	}
	fmt.Print(string(b))
}

func runRecoverChainIdTemplate(cmd *cobra.Command, args []string) {
	if *module == "" {
		log.Fatal("--module must be specified.")
	}
	if *recoverChainIdEvmChainId == "" {
		log.Fatal("--evm-chain-id must be specified.")
	}
	if _, err := isValidUint256(*recoverChainIdEvmChainId); err != nil {
		log.Fatal("failed to parse evm chain id as uint256:", err)
	}
	if *recoverChainIdNewChainId == "" {
		log.Fatal("--new-chain-id must be specified.")
	}
	newChainID, err := parseChainID(*recoverChainIdNewChainId)
	if err != nil {
		log.Fatal("failed to parse chain id:", err)
	}

	m := &nodev1.InjectGovernanceVAARequest{
		CurrentSetIndex: uint32(*templateGuardianIndex), // #nosec G115 -- This will never overflow
		Messages: []*nodev1.GovernanceMessage{
			{
				Sequence: rand.Uint64(),
				Nonce:    rand.Uint32(),
				Payload: &nodev1.GovernanceMessage_RecoverChainId{
					RecoverChainId: &nodev1.RecoverChainId{
						Module:     *module,
						EvmChainId: *recoverChainIdEvmChainId,
						NewChainId: uint32(newChainID),
					},
				},
			},
		},
	}

	b, err := prototext.MarshalOptions{Multiline: true}.Marshal(m)
	if err != nil {
		log.Fatal("failed to marshal request: ", err)
	}
	fmt.Print(string(b))
}

func runAccountantModifyBalanceTemplate(cmd *cobra.Command, args []string) {
	if *accountantModifyBalanceModule == "" {
		log.Fatal("--module must be specified.")
	}
	if *accountantModifyBalanceTargetChainId == "" {
		log.Fatal("--target-chain-id must be specified.")
	}
	targetChainID, err := parseChainID(*accountantModifyBalanceTargetChainId)
	if err != nil {
		log.Fatal("failed to parse target chain id: ", err)
	}
	if *accountantModifyBalanceSequence == "" {
		log.Fatal("--sequence must be specified")
	}
	sequence, err := strconv.ParseUint(*accountantModifyBalanceSequence, 10, 64)
	if err != nil {
		log.Fatal("failed to parse sequence as uint64: ", err)
	}
	if *accountantModifyBalanceChainId == "" {
		log.Fatal("--chain-id must be specified.")
	}
	chainID, err := parseChainID(*accountantModifyBalanceChainId)
	if err != nil {
		log.Fatal("failed to parse chain id: ", err)
	}
	if *accountantModifyBalanceTokenChainId == "" {
		log.Fatal("--token-chain-id must be specified.")
	}
	tokenChainID, err := parseChainID(*accountantModifyBalanceTokenChainId)
	if err != nil {
		log.Fatal("failed to parse token chain id: ", err)
	}
	if *accountantModifyBalanceTokenAddress == "" {
		log.Fatal("--token-address must be specified.")
	}
	tokenAddress, err := parseAddress(*accountantModifyBalanceTokenAddress)
	if err != nil {
		log.Fatal("failed to parse token address: ", err)
	}
	if *accountantModifyBalanceAction == "" {
		log.Fatal("--action must be specified")
	}
	action, err := strconv.ParseUint(*accountantModifyBalanceAction, 10, 8)
	if err != nil {
		log.Fatal("failed to parse modification action as uint8: ", err)
	}
	if action != uint64(nodev1.ModificationKind_MODIFICATION_KIND_ADD) && action != uint64(nodev1.ModificationKind_MODIFICATION_KIND_SUBTRACT) {
		log.Fatal("invalid modification action, must be 1 (add) or 2 (subtract)")
	}
	if *accountantModifyBalanceAmount == "" {
		log.Fatal("--amount must be specified.")
	}
	amount_big := big.NewInt(0)
	amount_big, ok := amount_big.SetString(*accountantModifyBalanceAmount, 10)
	if !ok {
		log.Fatal("failed to parse amount")
	}
	_, overflow := uint256.FromBig(amount_big)
	if overflow {
		log.Fatal("amount overflowed uint256")
	}
	if *accountantModifyBalanceReason == "" {
		log.Fatal("--reason must be specified.")
	}
	if len(*accountantModifyBalanceReason) > vaa.AccountantModifyBalanceReasonLength {
		log.Fatalf("reason is too long, can be at most %d bytes", vaa.AccountantModifyBalanceReasonLength)
	}
	m := &nodev1.InjectGovernanceVAARequest{
		CurrentSetIndex: uint32(*templateGuardianIndex), // #nosec G115 -- This will never overflow
		Messages: []*nodev1.GovernanceMessage{
			{
				Sequence: rand.Uint64(),
				Nonce:    rand.Uint32(),
				Payload: &nodev1.GovernanceMessage_AccountantModifyBalance{
					AccountantModifyBalance: &nodev1.AccountantModifyBalance{
						Module:        *accountantModifyBalanceModule,
						TargetChainId: uint32(targetChainID),
						Sequence:      uint64(sequence),
						ChainId:       uint32(chainID),
						TokenChain:    uint32(tokenChainID),
						TokenAddress:  tokenAddress,
						Kind:          nodev1.ModificationKind(action),
						Amount:        *accountantModifyBalanceAmount,
						Reason:        *accountantModifyBalanceReason,
					},
				},
			},
		},
	}

	b, err := prototext.MarshalOptions{Multiline: true}.Marshal(m)
	if err != nil {
		log.Fatal("failed to marshal request: ", err)
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
		CurrentSetIndex: uint32(*templateGuardianIndex), // #nosec G115 -- This will never overflow
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
		log.Fatal("failed to marshal request: ", err)
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
		CurrentSetIndex: uint32(*templateGuardianIndex), // #nosec G115 -- This will never overflow
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
		log.Fatal("failed to marshal request: ", err)
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
		CurrentSetIndex: uint32(*templateGuardianIndex), // #nosec G115 -- This will never overflow
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
		log.Fatal("failed to marshal request: ", err)
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
		CurrentSetIndex: uint32(*templateGuardianIndex), // #nosec G115 -- This will never overflow
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
		log.Fatal("failed to marshal request: ", err)
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
		CurrentSetIndex: uint32(*templateGuardianIndex), // #nosec G115 -- This will never overflow
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
		log.Fatal("failed to marshal request: ", err)
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
		log.Fatal("--instantiation-msg must be specified.")
	}

	m := &nodev1.InjectGovernanceVAARequest{
		CurrentSetIndex: uint32(*templateGuardianIndex), // #nosec G115 -- This will never overflow
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
		log.Fatal("failed to marshal request: ", err)
	}
	fmt.Print(string(b))
}

func runWormchainAddWasmInstantiateAllowlistTemplate(cmd *cobra.Command, args []string) {
	runWormchainWasmInstantiateAllowlistTemplate(nodev1.WormchainWasmInstantiateAllowlistAction_WORMCHAIN_WASM_INSTANTIATE_ALLOWLIST_ACTION_ADD)
}

func runWormchainDeleteWasmInstantiateAllowlistTemplate(cmd *cobra.Command, args []string) {
	runWormchainWasmInstantiateAllowlistTemplate(nodev1.WormchainWasmInstantiateAllowlistAction_WORMCHAIN_WASM_INSTANTIATE_ALLOWLIST_ACTION_DELETE)
}

func runWormchainWasmInstantiateAllowlistTemplate(action nodev1.WormchainWasmInstantiateAllowlistAction) {
	if *wormchainWasmInstantiateAllowlistCodeId == "" {
		log.Fatal("--code-id must be specified")
	}
	codeId, err := strconv.ParseUint(*wormchainWasmInstantiateAllowlistCodeId, 10, 64)
	if err != nil {
		log.Fatal("failed to parse code-id as utin64: ", err)
	}
	if *wormchainWasmInstantiateAllowlistContractAddress == "" {
		log.Fatal("--contract-address must be specified")
	}

	m := &nodev1.InjectGovernanceVAARequest{
		CurrentSetIndex: uint32(*templateGuardianIndex), // #nosec G115 -- This will never overflow
		Messages: []*nodev1.GovernanceMessage{
			{
				Sequence: rand.Uint64(),
				Nonce:    rand.Uint32(),
				Payload: &nodev1.GovernanceMessage_WormchainWasmInstantiateAllowlist{
					WormchainWasmInstantiateAllowlist: &nodev1.WormchainWasmInstantiateAllowlist{
						CodeId:   codeId,
						Contract: *wormchainWasmInstantiateAllowlistContractAddress,
						Action:   action,
					},
				},
			},
		},
	}

	b, err := prototext.MarshalOptions{Multiline: true}.Marshal(m)
	if err != nil {
		log.Fatal("failed to marshal request: ", err)
	}
	fmt.Print(string(b))
}

func runGatewayScheduleUpgradeTemplate(cmd *cobra.Command, args []string) {
	if *gatewayScheduleUpgradeName == "" {
		log.Fatal("--name must be specified")
	}

	if *gatewayScheduleUpgradeHeight == "" {
		log.Fatal("--height must be specified")
	}

	height, err := strconv.ParseUint(*gatewayScheduleUpgradeHeight, 10, 64)
	if err != nil {
		log.Fatal("failed to parse height as uint64: ", err)
	}

	m := &nodev1.InjectGovernanceVAARequest{
		CurrentSetIndex: uint32(*templateGuardianIndex), // #nosec G115 -- This will never overflow
		Messages: []*nodev1.GovernanceMessage{
			{
				Sequence: rand.Uint64(),
				Nonce:    rand.Uint32(),
				Payload: &nodev1.GovernanceMessage_GatewayScheduleUpgrade{
					GatewayScheduleUpgrade: &nodev1.GatewayScheduleUpgrade{
						Name:   *gatewayScheduleUpgradeName,
						Height: height,
					},
				},
			},
		},
	}

	b, err := prototext.MarshalOptions{Multiline: true}.Marshal(m)
	if err != nil {
		log.Fatal("failed to marshal request: ", err)
	}
	fmt.Print(string(b))
}

func runGatewayCancelUpgradeTemplate(cmd *cobra.Command, args []string) {
	m := &nodev1.InjectGovernanceVAARequest{
		CurrentSetIndex: uint32(*templateGuardianIndex), // #nosec G115 -- This will never overflow
		Messages: []*nodev1.GovernanceMessage{
			{
				Sequence: rand.Uint64(),
				Nonce:    rand.Uint32(),
				Payload:  &nodev1.GovernanceMessage_GatewayCancelUpgrade{},
			},
		},
	}

	b, err := prototext.MarshalOptions{Multiline: true}.Marshal(m)
	if err != nil {
		log.Fatal("failed to marshal request: ", err)
	}
	fmt.Print(string(b))
}

func runGatewayIbcComposabilityMwSetContractTemplate(cmd *cobra.Command, args []string) {
	if *gatewayIbcComposabilityMwContractAddress == "" {
		log.Fatal("--contract-address must be specified")
	}

	m := &nodev1.InjectGovernanceVAARequest{
		CurrentSetIndex: uint32(*templateGuardianIndex), // #nosec G115 -- This will never overflow
		Messages: []*nodev1.GovernanceMessage{
			{
				Sequence: rand.Uint64(),
				Nonce:    rand.Uint32(),
				Payload: &nodev1.GovernanceMessage_GatewayIbcComposabilityMwSetContract{
					GatewayIbcComposabilityMwSetContract: &nodev1.GatewayIbcComposabilityMwSetContract{
						Contract: *gatewayIbcComposabilityMwContractAddress,
					},
				},
			},
		},
	}

	b, err := prototext.MarshalOptions{Multiline: true}.Marshal(m)
	if err != nil {
		log.Fatal("failed to marshal request: ", err)
	}
	fmt.Print(string(b))
}

func runIbcReceiverUpdateChannelChainTemplate(cmd *cobra.Command, args []string) {
	runIbcUpdateChannelChainTemplate(nodev1.IbcUpdateChannelChainModule_IBC_UPDATE_CHANNEL_CHAIN_MODULE_RECEIVER)
}

func runIbcTranslatorUpdateChannelChainTemplate(cmd *cobra.Command, args []string) {
	runIbcUpdateChannelChainTemplate(nodev1.IbcUpdateChannelChainModule_IBC_UPDATE_CHANNEL_CHAIN_MODULE_TRANSLATOR)
}

func runIbcUpdateChannelChainTemplate(module nodev1.IbcUpdateChannelChainModule) {
	if *ibcUpdateChannelChainTargetChainId == "" {
		log.Fatal("--target-chain-id must be specified")
	}
	targetChainId, err := parseChainID(*ibcUpdateChannelChainTargetChainId)
	if err != nil {
		log.Fatal("failed to parse chain id: ", err)
	}

	if *ibcUpdateChannelChainChannelId == "" {
		log.Fatal("--channel-id must be specified")
	}
	if len(*ibcUpdateChannelChainChannelId) > 64 {
		log.Fatal("invalid channel id length, must be <= 64")
	}

	if *ibcUpdateChannelChainChainId == "" {
		log.Fatal("--chain-id must be specified")
	}
	chainId, err := parseChainID(*ibcUpdateChannelChainChainId)
	if err != nil {
		log.Fatal("failed to parse chain id: ", err)
	}

	m := &nodev1.InjectGovernanceVAARequest{
		CurrentSetIndex: uint32(*templateGuardianIndex), // #nosec G115 -- This will never overflow
		Messages: []*nodev1.GovernanceMessage{
			{
				Sequence: rand.Uint64(),
				Nonce:    rand.Uint32(),
				Payload: &nodev1.GovernanceMessage_IbcUpdateChannelChain{
					IbcUpdateChannelChain: &nodev1.IbcUpdateChannelChain{
						TargetChainId: uint32(targetChainId),
						ChannelId:     *ibcUpdateChannelChainChannelId,
						ChainId:       uint32(chainId),
						Module:        module,
					},
				},
			},
		},
	}

	b, err := prototext.MarshalOptions{Multiline: true}.Marshal(m)
	if err != nil {
		log.Fatal("failed to marshal request: ", err)
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
		CurrentSetIndex: uint32(*templateGuardianIndex), // #nosec G115 -- This will never overflow
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
		log.Fatal("failed to marshal request: ", err)
	}
	fmt.Print(string(b))
}

func runGeneralPurposeGovernanceEvmCallTemplate(cmd *cobra.Command, args []string) {
	if *governanceTargetAddress == "" {
		log.Fatal("--target-address must be specified")
	}
	if !common.IsHexAddress(*governanceTargetAddress) {
		log.Fatal("invalid target address")
	}
	governanceTargetAddress := common.HexToAddress(*governanceTargetAddress).Hex()
	if *governanceCallData == "" {
		log.Fatal("--call-data must be specified")
	}
	if *governanceContractAddress == "" {
		log.Fatal("--governance-contract must be specified")
	}
	if !common.IsHexAddress(*governanceContractAddress) {
		log.Fatal("invalid governance contract address")
	}
	governanceContractAddress := common.HexToAddress(*governanceContractAddress).Hex()
	if *governanceTargetChain == "" {
		log.Fatal("--chain-id must be specified")
	}
	chainID, err := parseChainID(*governanceTargetChain)
	if err != nil {
		log.Fatal("failed to parse chain id: ", err)
	}

	m := &nodev1.InjectGovernanceVAARequest{
		CurrentSetIndex: uint32(*templateGuardianIndex), // #nosec G115 -- This will never overflow
		Messages: []*nodev1.GovernanceMessage{
			{
				Sequence: rand.Uint64(),
				Nonce:    rand.Uint32(),
				Payload: &nodev1.GovernanceMessage_EvmCall{
					EvmCall: &nodev1.EvmCall{
						ChainId:            uint32(chainID),
						GovernanceContract: governanceContractAddress,
						TargetContract:     governanceTargetAddress,
						AbiEncodedCall:     *governanceCallData,
					},
				},
			},
		},
	}

	b, err := prototext.MarshalOptions{Multiline: true}.Marshal(m)
	if err != nil {
		log.Fatal("failed to marshal request: ", err)
	}
	fmt.Print(string(b))
}

func runGeneralPurposeGovernanceSolanaCallTemplate(cmd *cobra.Command, args []string) {
	if *governanceCallData == "" {
		log.Fatal("--call-data must be specified")
	}
	if *governanceContractAddress == "" {
		log.Fatal("--governance-contract must be specified")
	}
	_, err := base58.Decode(*governanceContractAddress)
	if err != nil {
		log.Fatal("invalid base58 governance contract address")
	}
	if *governanceTargetChain == "" {
		log.Fatal("--chain-id must be specified")
	}
	chainID, err := parseChainID(*governanceTargetChain)
	if err != nil {
		log.Fatal("failed to parse chain id: ", err)
	}

	m := &nodev1.InjectGovernanceVAARequest{
		CurrentSetIndex: uint32(*templateGuardianIndex), // #nosec G115 -- This will never overflow
		Messages: []*nodev1.GovernanceMessage{
			{
				Sequence: rand.Uint64(),
				Nonce:    rand.Uint32(),
				Payload: &nodev1.GovernanceMessage_SolanaCall{
					SolanaCall: &nodev1.SolanaCall{
						ChainId:            uint32(chainID),
						GovernanceContract: *governanceContractAddress,
						EncodedInstruction: *governanceCallData,
					},
				},
			},
		},
	}

	b, err := prototext.MarshalOptions{Multiline: true}.Marshal(m)
	if err != nil {
		log.Fatal("failed to marshal request: ", err)
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
		return "", errors.New("address longer than 32 bytes")
	}
	return hex.EncodeToString(common.LeftPadBytes(a, 32)), nil
}

// parseChainID parses a human-readable chain name or a chain ID.
func parseChainID(name string) (vaa.ChainID, error) {
	s, err := vaa.ChainIDFromString(name)
	if err == nil {
		return s, nil
	}

	// parse as uint16
	i, err := strconv.ParseUint(name, 10, 16)
	if err != nil {
		return 0, fmt.Errorf("failed to parse as name or uint16: %v", err)
	}

	return vaa.ChainID(i), nil
}

func isValidUint256(s string) (bool, error) {
	i := new(big.Int)
	i.SetString(s, 10) // Parse in base 10

	// Create upper limit as 2^256 - 1
	upperLimit := new(big.Int)
	upperLimit.Exp(big.NewInt(2), big.NewInt(256), nil)
	upperLimit.Sub(upperLimit, big.NewInt(1))

	// Check if i is within the range [0, 2^256 - 1]
	if i.Cmp(big.NewInt(0)) < 0 || i.Cmp(upperLimit) > 0 {
		return false, errors.New("value is not a valid uint256")
	}

	return true, nil
}
