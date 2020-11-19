package guardiand

import (
	"io/ioutil"
	"log"

	"github.com/davecgh/go-spew/spew"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/encoding/prototext"

	nodev1 "github.com/certusone/wormhole/bridge/pkg/proto/node/v1"
)

var AdminClientGuardianSetVerifyCmd = &cobra.Command{
	Use:   "guardian-set-update-verify",
	Short: "Verify guardian set update in prototxt format (offline)",
	Run:   runGuardianSetVerify,
	Args:  cobra.ExactArgs(1),
}

func runGuardianSetVerify(cmd *cobra.Command, args []string) {
	path := args[0]

	b, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatalf("failed to read file: %v", err)
	}

	var msg nodev1.GuardianSetUpdate
	err = prototext.Unmarshal(b, &msg)
	if err != nil {
		log.Fatalf("failed to deserialize: %v", err)
	}

	v, err := adminGuardianSetUpdateToVAA(&msg)
	if err != nil {
		log.Fatalf("invalid update: %v", err)
	}

	digest, err := v.SigningMsg()
	if err != nil {
		panic(err)
	}

	log.Printf("VAA with digest %s: %+v", digest.Hex(), spew.Sdump(v))
}
