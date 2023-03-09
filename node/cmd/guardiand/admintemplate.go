package guardiand

import (
	"crypto/ecdsa"
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
var shutdownGuardianKey *string
var shutdownPubKey *string

var circleIntegrationChainID *string
var circleIntegrationFinality *string
var circleIntegrationForeignEmitterChainID *string
var circleIntegrationForeignEmitterAddress *string
var circleIntegrationCircleDomain *string
var circleIntegrationNewImplementationAddress *string

func init() {
	governanceFlagSet := pflag.NewFlagSet("governance", pflag.ExitOnError)
	chainID = governanceFlagSet.String("chain-id", "", "Chain ID")
	address = governanceFlagSet.String("new-address", "", "New address (hex, base58 or bech32)")

	moduleFlagSet := pflag.NewFlagSet("module", pflag.ExitOnError)
	module = moduleFlagSet.String("module", "", "Module name")

	authProofFlagSet := pflag.NewFlagSet("auth-proof", pflag.ExitOnError)
	shutdownGuardianKey = authProofFlagSet.String("guardian-key", "", "Guardian key to sign proof. File path or hex string")
	shutdownPubKey = authProofFlagSet.String("proof-pub-key", "", "Public key to encode in proof")

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

	AdminClientWormholeRelayerSetDefaultRelayProviderCmd.Flags().AddFlagSet(governanceFlagSet)
	TemplateCmd.AddCommand(AdminClientWormholeRelayerSetDefaultRelayProviderCmd)

	AdminClientShutdownProofCmd.Flags().AddFlagSet(authProofFlagSet)
	TemplateCmd.AddCommand(AdminClientShutdownProofCmd)

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
var AdminClientShutdownProofCmd = &cobra.Command{
	Use:   "shutdown-proof",
	Short: "Generate an auth proof for shutdown voting on behalf of the guardian.",
	Run:   runShutdownProofTemplate,
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

var AdminClientWormholeRelayerSetDefaultRelayProviderCmd = &cobra.Command{
	Use: "wormhole-relayer-set-default-relay-provider",
	Short: "Generate a 'set default relay provider' template for specified chain and address",
	Run: runWormholeRelayerSetDefaultRelayProviderTemplate,
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

func runShutdownProofTemplate(cmd *cobra.Command, args []string) {
	// ensure values were passed
	if *shutdownPubKey == "" {
		log.Fatal("--proof-pub-key cannot be blank.")
	}
	if *shutdownGuardianKey == "" {
		log.Fatal("--guardian-key cannot be blank.")
	}

	// load the guardian key that will sign the proof
	var guardianKey *ecdsa.PrivateKey
	var keyErr error
	// check if the key is a hex string
	_, hexDecodeErr := hex.DecodeString(*shutdownGuardianKey)
	if hexDecodeErr == nil {
		guardianKey, keyErr = crypto.HexToECDSA(*shutdownGuardianKey)
	} else {
		// the supplied guardian key is not hex, must be a file path to load
		guardianKey, keyErr = loadGuardianKey(*shutdownGuardianKey)
	}
	if keyErr != nil {
		log.Fatal("failed fetching guardian key.", keyErr)
	}

	// create the payload of the proof
	pubKey := common.HexToAddress(*shutdownPubKey)
	digest := crypto.Keccak256Hash(pubKey.Bytes())

	// sign the payload of the proof
	ethProof, err := crypto.Sign(digest.Bytes(), guardianKey)
	if err != nil {
		log.Fatal("failed creating proof.", err)
	}

	// log the public key in the proof and the public key that signed the proof
	fmt.Printf(
		"The following proof will allow public key \"%v\" to vote on behalf of guardian \"%v\":\n",
		pubKey.Hex(),
		crypto.PubkeyToAddress(guardianKey.PublicKey),
	)

	proofHex := hex.EncodeToString(ethProof)
	fmt.Print(proofHex)
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

func runWormholeRelayerSetDefaultRelayProviderTemplate(cmd *cobra.Command, args []string) {
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
				Payload: &nodev1.GovernanceMessage_WormholeRelayerSetDefaultRelayProvider{
					WormholeRelayerSetDefaultRelayProvider: &nodev1.WormholeRelayerSetDefaultRelayProvider{
						ChainId:     uint32(chainID),
						NewDefaultRelayProviderAddress: address,
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
