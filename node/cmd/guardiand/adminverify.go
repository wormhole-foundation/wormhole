package guardiand

import (
	"encoding/hex"
	"log"
	"os"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/spf13/cobra"
	"github.com/status-im/keycard-go/hexutils"
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

	timestamp := time.Unix(int64(req.Timestamp), 0)

	for _, message := range req.Messages {
		v, err := adminrpc.GovMsgToVaa(message, req.CurrentSetIndex, timestamp)

		if err != nil {
			log.Fatalf("invalid update: %v", err)
		}

		digest := v.SigningDigest().Bytes()
		if err != nil {
			panic(err)
		}

		b, err := v.Marshal()
		if err != nil {
			panic(err)
		}

		log.Printf("Serialized: %v", hex.EncodeToString(b))

		log.Printf("VAA with digest %s: %+v", hexutils.BytesToHex(digest), spew.Sdump(v))
	}
}
