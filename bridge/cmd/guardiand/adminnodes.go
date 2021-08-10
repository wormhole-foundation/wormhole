package guardiand

import (
	"context"
	"fmt"
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

var (
	showDetails bool
)

func init() {
	AdminClientListNodes.Flags().BoolVar(&showDetails, "showDetails", false, "Show error counter and contract addresses")
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

	gs, err := c.GetCurrentGuardianSet(ctx, &publicrpcv1.GetCurrentGuardianSetRequest{})
	if err != nil {
		log.Fatalf("failed to list current guardian get: %v", err)
	}

	log.Printf("current guardian set index: %d (%d guardians)",
		gs.GuardianSet.Index, len(gs.GuardianSet.Addresses))

	nodes := lastHeartbeats.Entries

	sort.Slice(nodes, func(i, j int) bool {
		if nodes[i].RawHeartbeat == nil || nodes[j].RawHeartbeat == nil {
			return false
		}
		return nodes[i].RawHeartbeat.NodeName < nodes[j].RawHeartbeat.NodeName
	})

	log.Printf("%d nodes in guardian state set", len(nodes))

	w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)

	if showDetails {
		w.Write([]byte("Node key\tGuardian key\tNode name\tVersion\tLast seen\tUptime\tSolana\tEthereum\tTerra\tBSC\n"))
	} else {
		w.Write([]byte("Node key\tGuardian key\tNode name\tVersion\tLast seen\tSolana\tEthereum\tTerra\tBSC\n"))
	}

	for _, h := range nodes {
		if h.RawHeartbeat == nil {
			continue
		}

		last := time.Unix(0, h.RawHeartbeat.Timestamp)
		boot := time.Unix(0, h.RawHeartbeat.BootTimestamp)

		heights := map[vaa.ChainID]int64{}
		truncAddrs := make(map[vaa.ChainID]string)
		errors := map[vaa.ChainID]uint64{}
		for _, n := range h.RawHeartbeat.Networks {
			heights[vaa.ChainID(n.Id)] = n.Height
			errors[vaa.ChainID(n.Id)] = n.ErrorCount
			if len(n.BridgeAddress) >= 16 {
				truncAddrs[vaa.ChainID(n.Id)] = n.BridgeAddress[:16]
			} else {
				truncAddrs[vaa.ChainID(n.Id)] = "INVALID"
			}
		}

		if showDetails {
			fmt.Fprintf(w,
				"%s\t%s\t%s\t%s\t%s\t%s\t%s %d (%d)\t%s %d (%d)\t%s %d (%d)\t%s %d (%d)\n",
				h.P2PNodeAddr,
				h.RawHeartbeat.GuardianAddr,
				h.RawHeartbeat.NodeName,
				h.RawHeartbeat.Version,
				time.Since(last),
				time.Since(boot),
				truncAddrs[vaa.ChainIDSolana],
				heights[vaa.ChainIDSolana],
				errors[vaa.ChainIDSolana],
				truncAddrs[vaa.ChainIDEthereum],
				heights[vaa.ChainIDEthereum],
				errors[vaa.ChainIDEthereum],
				truncAddrs[vaa.ChainIDTerra],
				heights[vaa.ChainIDTerra],
				errors[vaa.ChainIDTerra],
				truncAddrs[vaa.ChainIDBSC],
				heights[vaa.ChainIDBSC],
				errors[vaa.ChainIDBSC],
			)
		} else {
			fmt.Fprintf(w,
				"%s\t%s\t%s\t%s\t%s\t%d\t%d\t%d\t%d\n",
				h.P2PNodeAddr,
				h.RawHeartbeat.GuardianAddr,
				h.RawHeartbeat.NodeName,
				h.RawHeartbeat.Version,
				time.Since(last),
				heights[vaa.ChainIDSolana],
				heights[vaa.ChainIDEthereum],
				heights[vaa.ChainIDTerra],
				heights[vaa.ChainIDBSC],
			)
		}
	}

	w.Flush()
	fmt.Print("\n")

	for _, addr := range gs.GuardianSet.Addresses {
		var found bool
		for _, h := range nodes {
			if h.VerifiedGuardianAddr == addr {
				found = true
			}
		}

		if !found {
			fmt.Printf("Missing guardian: %s\n", addr)
		}
	}
}
