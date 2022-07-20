package guardiand

import (
	"context"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	publicrpcv1 "github.com/certusone/wormhole/node/pkg/proto/publicrpc/v1"
	"github.com/certusone/wormhole/node/pkg/vaa"
	"github.com/spf13/cobra"
)

// How to test in container:
//    kubectl exec guardian-0 -- /guardiand admin list-nodes --socket /tmp/admin.sock

var (
	showDetails bool
	only        []string
)

func init() {
	AdminClientListNodes.Flags().BoolVar(&showDetails, "showDetails", false, "Show error counter and contract addresses")
	AdminClientListNodes.Flags().StringSliceVar(&only, "only", nil, "Show only networks with the given name")
}

var AdminClientListNodes = &cobra.Command{
	Use:   "list-nodes",
	Short: "Fetches an aggregated list of guardian nodes",
	Run:   runListNodes,
}

func runListNodes(cmd *cobra.Command, args []string) {
	ctx := context.Background()
	conn, c, err := getPublicRPCServiceClient(ctx, *clientSocketPath)
	if err != nil {
		log.Fatalf("failed to get publicrpc client: %v", err)
	}
	defer conn.Close()

	lastHeartbeats, err := c.GetLastHeartbeats(ctx, &publicrpcv1.GetLastHeartbeatsRequest{})
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

	// Check if any node is sending Ropsten metrics
	var isTestnet bool
	for _, node := range nodes {
		for _, network := range node.RawHeartbeat.Networks {
			if vaa.ChainID(network.Id) == vaa.ChainIDEthereumRopsten {
				isTestnet = true
			}
		}
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)

	headers := []string{
		"Node key",
		"Guardian key",
		"Node name",
		"Version",
		"Last seen",
	}

	if showDetails {
		headers = append(headers, "Uptime")
	}

	type network struct {
		string
		vaa.ChainID
	}

	networks := []network{
		{"Solana", vaa.ChainIDSolana},
		{"Ethereum", vaa.ChainIDEthereum},
		{"Terra", vaa.ChainIDTerra},
		{"BSC", vaa.ChainIDBSC},
		{"Polygon", vaa.ChainIDPolygon},
		{"Avalanche", vaa.ChainIDAvalanche},
		{"Algorand", vaa.ChainIDAlgorand},
		{"Oasis", vaa.ChainIDOasis},
		{"Aurora", vaa.ChainIDAurora},
		{"Fantom", vaa.ChainIDFantom},
		{"Karura", vaa.ChainIDKarura},
		{"Acala", vaa.ChainIDAcala},
		{"Klaytn", vaa.ChainIDKlaytn},
		{"Celo", vaa.ChainIDCelo},
		{"Terra2", vaa.ChainIDTerra2},
	}

	if isTestnet {
		networks = append(networks, network{"Ropsten", vaa.ChainIDEthereumRopsten})
		networks = append(networks, network{"Moonbeam", vaa.ChainIDMoonbeam})
		networks = append(networks, network{"Neon", vaa.ChainIDNeon})
		networks = append(networks, network{"Injective", vaa.ChainIDInjective})
	}

	if len(only) > 0 {
		var filtered []network
		for _, network := range networks {
			for _, name := range only {
				if strings.EqualFold(network.string, name) {
					filtered = append(filtered, network)
				}
			}
		}
		networks = filtered
	}

	for _, k := range networks {
		headers = append(headers, k.string)
	}

	for _, header := range headers {
		_, _ = fmt.Fprintf(w, "%s\t", header)
	}
	_, _ = fmt.Fprintln(w)

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
			if len(n.ContractAddress) >= 16 {
				truncAddrs[vaa.ChainID(n.Id)] = n.ContractAddress[:16]
			} else {
				truncAddrs[vaa.ChainID(n.Id)] = "INVALID"
			}
		}

		fields := []string{
			h.P2PNodeAddr,
			h.RawHeartbeat.GuardianAddr,
			h.RawHeartbeat.NodeName,
			h.RawHeartbeat.Version,
			time.Since(last).String(),
		}

		if showDetails {
			fields = append(fields, time.Since(boot).String())
		}

		for _, n := range networks {
			if showDetails {
				fields = append(fields, fmt.Sprintf("%s %d (%d)",
					truncAddrs[n.ChainID], heights[n.ChainID], errors[n.ChainID]))
			} else {
				fields = append(fields, fmt.Sprintf("%d", heights[n.ChainID]))
			}
		}

		for _, field := range fields {
			_, _ = fmt.Fprintf(w, "%s\t", field)
		}

		_, _ = fmt.Fprintln(w)
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

	fmt.Println("\n[do not parse - use the gRPC or REST API for scripting]")
}
