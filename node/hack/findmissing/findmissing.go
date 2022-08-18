package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	nodev1 "github.com/certusone/wormhole/node/pkg/proto/node/v1"
	"github.com/wormhole-foundation/wormhole/sdk"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	adminRPC       = flag.String("adminRPC", "/run/guardiand/admin.socket", "Admin RPC address")
	shouldBackfill = flag.Bool("backfill", true, "Backfill missing sequences")
	onlyChain      = flag.String("only", "", "Only check this chain")
)

func getAdminClient(ctx context.Context, addr string) (*grpc.ClientConn, error, nodev1.NodePrivilegedServiceClient) {
	conn, err := grpc.DialContext(ctx, fmt.Sprintf("unix:///%s", addr), grpc.WithTransportCredentials(insecure.NewCredentials()))

	if err != nil {
		log.Fatalf("failed to connect to %s: %v", addr, err)
	}

	c := nodev1.NewNodePrivilegedServiceClient(conn)
	return conn, err, c
}

func main() {
	flag.Parse()

	ctx := context.Background()

	conn, err, admin := getAdminClient(ctx, *adminRPC)
	defer conn.Close()
	if err != nil {
		log.Fatalf("failed to get admin client: %v", err)
	}

	var only vaa.ChainID
	if *onlyChain != "" {
		only, err = vaa.ChainIDFromString(*onlyChain)
		if err != nil {
			log.Fatalf("failed to parse chain id: %v", err)
		}
	}

	for _, emitter := range sdk.KnownEmitters {
		if only != vaa.ChainIDUnset {
			if emitter.ChainID != only {
				continue
			}
		}

		log.Printf("requesting missing sequences for %v %s", emitter.ChainID, emitter.Emitter)

		msg := nodev1.FindMissingMessagesRequest{
			EmitterChain:   uint32(emitter.ChainID),
			EmitterAddress: emitter.Emitter,
			RpcBackfill:    *shouldBackfill,
			BackfillNodes:  sdk.PublicRPCEndpoints,
		}
		resp, err := admin.FindMissingMessages(ctx, &msg)
		if err != nil {
			log.Fatalf("failed to run find FindMissingMessages RPC: %v", err)
		}

		for _, id := range resp.MissingMessages {
			fmt.Println(id)
		}
	}
}
