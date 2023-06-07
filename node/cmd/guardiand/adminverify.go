package guardiand

import (
	"log"
	"os"

	"github.com/spf13/cobra"
	"google.golang.org/protobuf/encoding/prototext"

	"github.com/certusone/wormhole/node/pkg/adminrpc"
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
	b, err := os.ReadFile(path)
	if err != nil {
		log.Fatalf("failed to read file: %v", err)
	}
	var req nodev1.InjectGovernanceVAARequest
	err = prototext.Unmarshal(b, &req)
	if err != nil {
		log.Fatalf("failed to deserialize: %v", err)
	}

	adminrpc.VerifyReq(&req)
}
