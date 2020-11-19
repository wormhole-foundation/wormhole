package guardiand

import (
	"fmt"
	"io/ioutil"
	"log"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/encoding/prototext"

	"github.com/certusone/wormhole/bridge/pkg/devnet"
	nodev1 "github.com/certusone/wormhole/bridge/pkg/proto/node/v1"
)

var templateNumGuardians *int

func init() {
	templateNumGuardians = AdminClientGuardianSetTemplateCmd.Flags().Int("num", 1, "Number of devnet guardians in example file")
}

var AdminClientGuardianSetTemplateCmd = &cobra.Command{
	Use:   "guardian-set-update-template",
	Short: "Generate an empty guardian set template at specified path (offline)",
	Run:   runGuardianSetTemplate,
	Args:  cobra.ExactArgs(1),
}

func runGuardianSetTemplate(cmd *cobra.Command, args []string) {
	path := args[0]

	// Use deterministic devnet addresses as examples in the template, such that this doubles as a test fixture.
	guardians := make([]*nodev1.GuardianSetUpdate_Guardian, *templateNumGuardians)
	for i := 0; i < *templateNumGuardians; i++ {
		k := devnet.DeterministicEcdsaKeyByIndex(crypto.S256(), uint64(i))
		guardians[i] = &nodev1.GuardianSetUpdate_Guardian{
			Pubkey: crypto.PubkeyToAddress(k.PublicKey).Hex(),
			Name:   fmt.Sprintf("Example validator %d", i),
		}
	}

	m := &nodev1.GuardianSetUpdate{
		CurrentSetIndex: 1,
		// Timestamp is hardcoded to make it reproducible on different devnet nodes.
		// In production, a real UNIX timestamp should be used (see node.proto).
		Timestamp: 1605744545,
		Guardians: guardians,
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
