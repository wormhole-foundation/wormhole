package guardiand

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"time"

	"github.com/spf13/cobra"
	"github.com/status-im/keycard-go/hexutils"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/prototext"

	nodev1 "github.com/certusone/wormhole/bridge/pkg/proto/node/v1"
)

var clientSocketPath *string

func init() {
	pf := AdminClientInjectGuardianSetUpdateCmd.Flags()
	clientSocketPath = pf.String("socket", "", "gRPC server path to connect to (usually unix:///path/to.sock)")
	err := cobra.MarkFlagRequired(pf, "socket")
	if err != nil {
		panic(err)
	}

	AdminCmd.AddCommand(AdminClientInjectGuardianSetUpdateCmd)
	AdminCmd.AddCommand(AdminClientGuardianSetTemplateCmd)
	AdminCmd.AddCommand(AdminClientGuardianSetVerifyCmd)
}

var AdminCmd = &cobra.Command{
	Use:   "admin",
	Short: "Guardian node admin commands",
}

var AdminClientInjectGuardianSetUpdateCmd = &cobra.Command{
	Use:   "guardian-set-update-inject",
	Short: "Inject and sign a guardian set update from a prototxt file (see docs!)",
	Run:   runInjectGuardianSetUpdate,
	Args:  cobra.ExactArgs(1),
}

func getAdminClient(ctx context.Context, addr string) (*grpc.ClientConn, error, nodev1.NodePrivilegedClient) {
	conn, err := grpc.DialContext(ctx,
		fmt.Sprintf("passthrough:///%s", addr),
		grpc.WithInsecure(),
		grpc.WithContextDialer(func(ctx context.Context, addr string) (net.Conn, error) {
			return new(net.Dialer).DialContext(ctx, "unix", addr)
		}))

	if err != nil {
		log.Fatalf("failed to connect to %s: %v", addr, err)
	}

	c := nodev1.NewNodePrivilegedClient(conn)
	return conn, err, c
}

func runInjectGuardianSetUpdate(cmd *cobra.Command, args []string) {
	path := args[0]
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err, c := getAdminClient(ctx, *clientSocketPath)
	defer conn.Close()

	b, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatalf("failed to read file: %v", err)
	}

	var msg nodev1.GuardianSetUpdate
	err = prototext.Unmarshal(b, &msg)
	if err != nil {
		log.Fatalf("failed to deserialize: %v", err)
	}

	resp, err := c.SubmitGuardianSetVAA(ctx, &nodev1.SubmitGuardianSetVAARequest{GuardianSet: &msg})
	if err != nil {
		log.Fatalf("failed to submit guardian set update: %v", err)
	}

	log.Printf("VAA successfully injected with digest %s", hexutils.BytesToHex(resp.Digest))
}
