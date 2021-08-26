package guardiand

import (
	"github.com/certusone/wormhole/node/pkg/vaa"
	"io/ioutil"
	"log"

	"github.com/davecgh/go-spew/spew"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/encoding/prototext"

	nodev1 "github.com/certusone/wormhole/node/pkg/proto/node/v1"
)

var AdminClientGovernanceVAAVerifyCmd = &cobra.Command{
	Use:   "governance-vaa-verify [FILENAME]",
	Short: "Verify governance vaa in prototxt format (offline)",
	Run:   runGovernanceVAAVerify,
	Args:  cobra.ExactArgs(1),
}

func runGovernanceVAAVerify(cmd *cobra.Command, args []string) {
	path := args[0]

	b, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatalf("failed to read file: %v", err)
	}

	var msg nodev1.InjectGovernanceVAARequest
	err = prototext.Unmarshal(b, &msg)
	if err != nil {
		log.Fatalf("failed to deserialize: %v", err)
	}

	var (
		v *vaa.VAA
	)
	switch payload := msg.Payload.(type) {
	case *nodev1.InjectGovernanceVAARequest_GuardianSet:
		v, err = adminGuardianSetUpdateToVAA(payload.GuardianSet, msg.CurrentSetIndex, msg.Timestamp)
	case *nodev1.InjectGovernanceVAARequest_ContractUpgrade:
		v, err = adminContractUpgradeToVAA(payload.ContractUpgrade, msg.CurrentSetIndex, msg.Timestamp)
	}
	if err != nil {
		log.Fatalf("invalid update: %v", err)
	}

	digest, err := v.SigningMsg()
	if err != nil {
		panic(err)
	}

	log.Printf("VAA with digest %s: %+v", digest.Hex(), spew.Sdump(v))
}
