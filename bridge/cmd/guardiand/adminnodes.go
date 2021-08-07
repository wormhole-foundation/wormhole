package guardiand

import (
	"context"
	"fmt"
	publicrpcv1 "github.com/certusone/wormhole/bridge/pkg/proto/publicrpc/v1"
	"github.com/spf13/cobra"
	"io"
	"log"
	"os"
	"text/tabwriter"
)

// How to test in container:
//    kubectl exec guardian-0 -- /guardiand admin list-nodes --socket /tmp/admin.sock

var AdminClientListNodesStream = &cobra.Command{
	Use:   "list-nodes-stream",
	Short: "Listens to heartbeats and displays an aggregated real-time list of guardian nodes",
	Run:   runListNodesStream,
}

func runListNodesStream(cmd *cobra.Command, args []string) {
	ctx := context.Background()
	conn, err, c := getPublicrpcClient(ctx, *clientSocketPath)
	defer conn.Close()
	if err != nil {
		log.Fatalf("failed to get publicrpc client: %v", err)
	}

	stream, err := c.GetRawHeartbeats(ctx, &publicrpcv1.GetRawHeartbeatsRequest{})
	if err != nil {
		log.Fatalf("failed to stream heartbeats: %v", err)
	}

	log.Print("connected, streaming updates")

	seen := make(map[string]bool)
	w := tabwriter.NewWriter(os.Stdout, 20, 8, 1, '\t', 0)
	for {
		hb, err := stream.Recv()
		if err == io.EOF {
			log.Print("server closed connection, exiting")
			return
		} else if err != nil {
			log.Fatalf("error streaming updates: %v", err)
		}

		if seen[hb.GuardianAddr] {
			continue
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t\n", hb.GuardianAddr, hb.NodeName, hb.Version)
		w.Flush()
		seen[hb.GuardianAddr] = true
	}
}
