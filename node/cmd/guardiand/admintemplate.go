package guardiand

import (
	"fmt"
	"github.com/tendermint/tendermint/libs/rand"
	"io/ioutil"
	"log"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/encoding/prototext"

	"github.com/certusone/wormhole/node/pkg/devnet"
	nodev1 "github.com/certusone/wormhole/node/pkg/proto/node/v1"
)

var setUpdateNumGuardians *int
var templateGuardianIndex *int

func init() {
	templateGuardianIndex = TemplateCmd.PersistentFlags().Int("idx", 0, "Default current guardian set index")
	setUpdateNumGuardians = AdminClientGuardianSetTemplateCmd.Flags().Int("num", 1, "Number of devnet guardians in example file")

	TemplateCmd.AddCommand(AdminClientGuardianSetTemplateCmd)
	TemplateCmd.AddCommand(AdminClientContractUpgradeTemplateCmd)
	TemplateCmd.AddCommand(AdminClientTokenBridgeRegisterChainCmd)
	TemplateCmd.AddCommand(AdminClientTokenBridgeUpgradeContractCmd)
}

var TemplateCmd = &cobra.Command{
	Use:   "template",
	Short: "Guardian governance VAA template commands ",
}

var AdminClientGuardianSetTemplateCmd = &cobra.Command{
	Use:   "guardian-set-update [FILENAME]",
	Short: "Generate an empty guardian set template at specified path (offline)",
	Run:   runGuardianSetTemplate,
	Args:  cobra.ExactArgs(1),
}

var AdminClientContractUpgradeTemplateCmd = &cobra.Command{
	Use:   "contract-upgrade [FILENAME]",
	Short: "Generate an empty contract upgrade template at specified path (offline)",
	Run:   runContractUpgradeTemplate,
	Args:  cobra.ExactArgs(1),
}

var AdminClientTokenBridgeRegisterChainCmd = &cobra.Command{
	Use:   "token-bridge-register-chain [FILENAME]",
	Short: "Generate an empty token bridge chain registration template at specified path (offline)",
	Run:   runTokenBridgeRegisterChainTemplate,
	Args:  cobra.ExactArgs(1),
}

var AdminClientTokenBridgeUpgradeContractCmd = &cobra.Command{
	Use:   "token-bridge-upgrade-contract [FILENAME]",
	Short: "Generate an empty token bridge contract upgrade template at specified path (offline)",
	Run:   runTokenBridgeUpgradeContractTemplate,
	Args:  cobra.ExactArgs(1),
}

func runGuardianSetTemplate(cmd *cobra.Command, args []string) {
	path := args[0]

	// Use deterministic devnet addresses as examples in the template, such that this doubles as a test fixture.
	guardians := make([]*nodev1.GuardianSetUpdate_Guardian, *setUpdateNumGuardians)
	for i := 0; i < *setUpdateNumGuardians; i++ {
		k := devnet.DeterministicEcdsaKeyByIndex(crypto.S256(), uint64(i))
		guardians[i] = &nodev1.GuardianSetUpdate_Guardian{
			Pubkey: crypto.PubkeyToAddress(k.PublicKey).Hex(),
			Name:   fmt.Sprintf("Example validator %d", i),
		}
	}

	m := &nodev1.InjectGovernanceVAARequest{
		CurrentSetIndex: uint32(*templateGuardianIndex),
		Sequence:        1234,
		Nonce:           rand.Uint32(),
		Payload: &nodev1.InjectGovernanceVAARequest_GuardianSet{
			GuardianSet: &nodev1.GuardianSetUpdate{Guardians: guardians},
		},
	}

	b, err := prototext.MarshalOptions{Multiline: true}.Marshal(m)
	if err != nil {
		panic(err)
	}

	err = ioutil.WriteFile(path, b, 0640)
	if err != nil {
		log.Fatal(err)
	}
}

func runContractUpgradeTemplate(cmd *cobra.Command, args []string) {
	path := args[0]

	m := &nodev1.InjectGovernanceVAARequest{
		CurrentSetIndex: uint32(*templateGuardianIndex),
		Sequence:        rand.Uint64(),
		Nonce:           rand.Uint32(),
		Payload: &nodev1.InjectGovernanceVAARequest_ContractUpgrade{
			ContractUpgrade: &nodev1.ContractUpgrade{
				ChainId:     1,
				NewContract: "0000000000000000000000000290FB167208Af455bB137780163b7B7a9a10C16",
			},
		},
	}

	b, err := prototext.MarshalOptions{Multiline: true}.Marshal(m)
	if err != nil {
		panic(err)
	}

	err = ioutil.WriteFile(path, b, 0640)
	if err != nil {
		log.Fatal(err)
	}
}
func runTokenBridgeRegisterChainTemplate(cmd *cobra.Command, args []string) {
	path := args[0]

	m := &nodev1.InjectGovernanceVAARequest{
		CurrentSetIndex: uint32(*templateGuardianIndex),
		Sequence:        rand.Uint64(),
		Nonce:           rand.Uint32(),
		Payload: &nodev1.InjectGovernanceVAARequest_BridgeRegisterChain{
			BridgeRegisterChain: &nodev1.BridgeRegisterChain{
				Module:         "TokenBridge",
				ChainId:        5,
				EmitterAddress: "0000000000000000000000000290FB167208Af455bB137780163b7B7a9a10C16",
			},
		},
	}

	b, err := prototext.MarshalOptions{Multiline: true}.Marshal(m)
	if err != nil {
		panic(err)
	}

	err = ioutil.WriteFile(path, b, 0640)
	if err != nil {
		log.Fatal(err)
	}
}

func runTokenBridgeUpgradeContractTemplate(cmd *cobra.Command, args []string) {
	path := args[0]

	m := &nodev1.InjectGovernanceVAARequest{
		CurrentSetIndex: uint32(*templateGuardianIndex),
		Sequence:        rand.Uint64(),
		Nonce:           rand.Uint32(),
		Payload: &nodev1.InjectGovernanceVAARequest_BridgeContractUpgrade{
			BridgeContractUpgrade: &nodev1.BridgeUpgradeContract{
				Module:        "TokenBridge",
				TargetChainId: 5,
				NewContract:   "0000000000000000000000000290FB167208Af455bB137780163b7B7a9a10C16",
			},
		},
	}

	b, err := prototext.MarshalOptions{Multiline: true}.Marshal(m)
	if err != nil {
		panic(err)
	}

	err = ioutil.WriteFile(path, b, 0640)
	if err != nil {
		log.Fatal(err)
	}
}
