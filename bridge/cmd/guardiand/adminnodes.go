package guardiand

import (
	"context"
	"fmt"
	gossipv1 "github.com/certusone/wormhole/bridge/pkg/proto/gossip/v1"
	publicrpcv1 "github.com/certusone/wormhole/bridge/pkg/proto/publicrpc/v1"
	"github.com/certusone/wormhole/bridge/pkg/vaa"
	"github.com/spf13/cobra"
	"io"
	"log"
	"os"
	"sort"
	"text/tabwriter"
	"time"
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

var AdminClientListNodes = &cobra.Command{
	Use:   "list-nodes",
	Short: "Fetches an aggregated list of guardian nodes",
	Run:   runListNodes,
}

func runListNodes(cmd *cobra.Command, args []string) {
	ctx := context.Background()
	conn, err, c := getPublicrpcClient(ctx, *clientSocketPath)
	defer conn.Close()
	if err != nil {
		log.Fatalf("failed to get publicrpc client: %v", err)
	}

	lastHeartbeats, err := c.GetLastHeartbeats(ctx, &publicrpcv1.GetLastHeartbeatRequest{})
	if err != nil {
		log.Fatalf("failed to list nodes: %v", err)
	}

	nodes := make([]*gossipv1.Heartbeat, len(lastHeartbeats.RawHeartbeats))
	i := 0
	for _, v := range lastHeartbeats.RawHeartbeats {
		nodes[i] = v
		i += 1
	}
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].NodeName < nodes[j].NodeName
	})

	log.Printf("%d nodes in guardian state set", len(nodes))

	w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)

	w.Write([]byte("Guardian key\tNode name\tVersion\tLast seen\tUptime\tSolana\tEthereum\tTerra\tBSC\n"))

	for _, h := range nodes {
		last := time.Unix(0, h.Timestamp)

		heights := map[vaa.ChainID]int64{}
		for _, n := range h.Networks {
			heights[vaa.ChainID(n.Id)] = n.Height
		}

		fmt.Fprintf(w,
			"%s\t%s\t%s\t%s\t%d\t%d\t%d\t%d\t%d\n",
			h.GuardianAddr,
			h.NodeName,
			h.Version,
			time.Since(last),
			h.Counter,
			heights[vaa.ChainIDSolana],
			heights[vaa.ChainIDEthereum],
			heights[vaa.ChainIDTerra],
			heights[vaa.ChainIDBSC],
		)
	}

	w.Flush()
}
