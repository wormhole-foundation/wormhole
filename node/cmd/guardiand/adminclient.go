package guardiand

import (
	"context"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/mr-tron/base58"
	"github.com/spf13/pflag"

	"github.com/certusone/wormhole/node/pkg/common"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	publicrpcv1 "github.com/certusone/wormhole/node/pkg/proto/publicrpc/v1"
	"github.com/certusone/wormhole/node/pkg/vaa"

	"github.com/spf13/cobra"
	"github.com/status-im/keycard-go/hexutils"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/prototext"

	nodev1 "github.com/certusone/wormhole/node/pkg/proto/node/v1"
)

var (
	clientSocketPath *string
	shouldBackfill   *bool
)

func init() {
	// Shared flags for all admin commands
	pf := pflag.NewFlagSet("commonAdminFlags", pflag.ContinueOnError)
	clientSocketPath = pf.String("socket", "", "gRPC admin server socket to connect to")
	err := cobra.MarkFlagRequired(pf, "socket")
	if err != nil {
		panic(err)
	}

	shouldBackfill = AdminClientFindMissingMessagesCmd.Flags().Bool(
		"backfill", false, "backfill missing VAAs from public RPC")

	AdminClientInjectGuardianSetUpdateCmd.Flags().AddFlagSet(pf)
	AdminClientFindMissingMessagesCmd.Flags().AddFlagSet(pf)
	AdminClientListNodes.Flags().AddFlagSet(pf)
	DumpVAAByMessageID.Flags().AddFlagSet(pf)
	SendObservationRequest.Flags().AddFlagSet(pf)
	ClientChainGovernorStatusCmd.Flags().AddFlagSet(pf)
	ClientChainGovernorReloadCmd.Flags().AddFlagSet(pf)
	ClientChainGovernorDropPendingVAACmd.Flags().AddFlagSet(pf)
	ClientChainGovernorReleasePendingVAACmd.Flags().AddFlagSet(pf)

	AdminCmd.AddCommand(AdminClientInjectGuardianSetUpdateCmd)
	AdminCmd.AddCommand(AdminClientFindMissingMessagesCmd)
	AdminCmd.AddCommand(AdminClientGovernanceVAAVerifyCmd)
	AdminCmd.AddCommand(AdminClientListNodes)
	AdminCmd.AddCommand(DumpVAAByMessageID)
	AdminCmd.AddCommand(SendObservationRequest)
	AdminCmd.AddCommand(ClientChainGovernorStatusCmd)
	AdminCmd.AddCommand(ClientChainGovernorReloadCmd)
	AdminCmd.AddCommand(ClientChainGovernorDropPendingVAACmd)
	AdminCmd.AddCommand(ClientChainGovernorReleasePendingVAACmd)
}

var AdminCmd = &cobra.Command{
	Use:   "admin",
	Short: "Guardian node admin commands",
}

var AdminClientInjectGuardianSetUpdateCmd = &cobra.Command{
	Use:   "governance-vaa-inject [FILENAME]",
	Short: "Inject and sign a governance VAA from a prototxt file (see docs!)",
	Run:   runInjectGovernanceVAA,
	Args:  cobra.ExactArgs(1),
}

var AdminClientFindMissingMessagesCmd = &cobra.Command{
	Use:   "find-missing-messages [CHAIN_ID] [EMITTER_ADDRESS_HEX]",
	Short: "Find sequence number gaps for the given chain ID and emitter address",
	Run:   runFindMissingMessages,
	Args:  cobra.ExactArgs(2),
}

var DumpVAAByMessageID = &cobra.Command{
	Use:   "dump-vaa-by-message-id [MESSAGE_ID]",
	Short: "Retrieve a VAA by message ID (chain/emitter/seq) and decode and dump the VAA",
	Run:   runDumpVAAByMessageID,
	Args:  cobra.ExactArgs(1),
}

var SendObservationRequest = &cobra.Command{
	Use:   "send-observation-request [CHAIN_ID|CHAIN_NAME] [TX_HASH_HEX]",
	Short: "Broadcast an observation request for the given chain ID and chain-specific tx_hash",
	Run:   runSendObservationRequest,
	Args:  cobra.ExactArgs(2),
}

var ClientChainGovernorStatusCmd = &cobra.Command{
	Use:   "governor-status",
	Short: "Displays the status of the chain governor",
	Run:   runChainGovernorStatus,
	Args:  cobra.ExactArgs(0),
}

var ClientChainGovernorReloadCmd = &cobra.Command{
	Use:   "governor-reload",
	Short: "Clears the chain governor history and reloads it from the database",
	Run:   runChainGovernorReload,
	Args:  cobra.ExactArgs(0),
}

var ClientChainGovernorDropPendingVAACmd = &cobra.Command{
	Use:   "governor-drop-pending-vaa [VAA_ID]",
	Short: "Removes the specified VAA (chain/emitter/seq) from the chain governor pending list",
	Run:   runChainGovernorDropPendingVAA,
	Args:  cobra.ExactArgs(1),
}

var ClientChainGovernorReleasePendingVAACmd = &cobra.Command{
	Use:   "governor-release-pending-vaa [VAA_ID]",
	Short: "Releases the specified VAA (chain/emitter/seq) from the chain governor pending list, publishing it immediately",
	Run:   runChainGovernorReleasePendingVAA,
	Args:  cobra.ExactArgs(1),
}

func getAdminClient(ctx context.Context, addr string) (*grpc.ClientConn, nodev1.NodePrivilegedServiceClient, error) {
	conn, err := grpc.DialContext(ctx, fmt.Sprintf("unix:///%s", addr), grpc.WithInsecure())

	if err != nil {
		log.Fatalf("failed to connect to %s: %v", addr, err)
	}

	c := nodev1.NewNodePrivilegedServiceClient(conn)
	return conn, c, err
}

func getPublicRPCServiceClient(ctx context.Context, addr string) (*grpc.ClientConn, publicrpcv1.PublicRPCServiceClient, error) {
	conn, err := grpc.DialContext(ctx, fmt.Sprintf("unix:///%s", addr), grpc.WithInsecure())

	if err != nil {
		log.Fatalf("failed to connect to %s: %v", addr, err)
	}

	c := publicrpcv1.NewPublicRPCServiceClient(conn)
	return conn, c, err
}

func runInjectGovernanceVAA(cmd *cobra.Command, args []string) {
	path := args[0]
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, c, err := getAdminClient(ctx, *clientSocketPath)
	if err != nil {
		log.Fatalf("failed to get admin client: %v", err)
	}
	defer conn.Close()

	b, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatalf("failed to read file: %v", err)
	}

	var msg nodev1.InjectGovernanceVAARequest
	err = prototext.Unmarshal(b, &msg)
	if err != nil {
		log.Fatalf("failed to deserialize: %v", err)
	}

	resp, err := c.InjectGovernanceVAA(ctx, &msg)
	if err != nil {
		log.Fatalf("failed to submit governance VAA: %v", err)
	}

	for _, digest := range resp.Digests {
		log.Printf("VAA successfully injected with digest %s", hexutils.BytesToHex(digest))
	}
}

func runFindMissingMessages(cmd *cobra.Command, args []string) {
	chainID, err := strconv.Atoi(args[0])
	if err != nil {
		log.Fatalf("invalid chain ID: %v", err)
	}
	emitterAddress := args[1]

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	conn, c, err := getAdminClient(ctx, *clientSocketPath)
	if err != nil {
		log.Fatalf("failed to get admin client: %v", err)
	}
	defer conn.Close()

	msg := nodev1.FindMissingMessagesRequest{
		EmitterChain:   uint32(chainID),
		EmitterAddress: emitterAddress,
		RpcBackfill:    *shouldBackfill,
		BackfillNodes:  common.PublicRPCEndpoints,
	}
	resp, err := c.FindMissingMessages(ctx, &msg)
	if err != nil {
		log.Fatalf("failed to run find FindMissingMessages RPC: %v", err)
	}

	for _, id := range resp.MissingMessages {
		fmt.Println(id)
	}

	log.Printf("processed %s sequences %d to %d (%d gaps)",
		emitterAddress, resp.FirstSequence, resp.LastSequence, len(resp.MissingMessages))
}

// runDumpVAAByMessageID uses GetSignedVAA to request the given message,
// then decode and dump the VAA.
func runDumpVAAByMessageID(cmd *cobra.Command, args []string) {
	// Parse the {chain,emitter,seq} string.
	parts := strings.Split(args[0], "/")
	if len(parts) != 3 {
		log.Fatalf("invalid message ID: %s", args[0])
	}
	chainID, err := strconv.ParseUint(parts[0], 10, 32)
	if err != nil {
		log.Fatalf("invalid chain ID: %v", err)
	}
	emitterAddress := parts[1]
	seq, err := strconv.ParseUint(parts[2], 10, 64)
	if err != nil {
		log.Fatalf("invalid sequence number: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, c, err := getPublicRPCServiceClient(ctx, *clientSocketPath)
	if err != nil {
		log.Fatalf("failed to get public RPC service client: %v", err)
	}
	defer conn.Close()

	msg := publicrpcv1.GetSignedVAARequest{
		MessageId: &publicrpcv1.MessageID{
			EmitterChain:   publicrpcv1.ChainID(chainID),
			EmitterAddress: emitterAddress,
			Sequence:       seq,
		},
	}
	resp, err := c.GetSignedVAA(ctx, &msg)
	if err != nil {
		log.Fatalf("failed to run GetSignedVAA RPC: %v", err)
	}

	v, err := vaa.Unmarshal(resp.VaaBytes)
	if err != nil {
		log.Fatalf("failed to decode VAA: %v", err)
	}

	log.Printf("VAA with digest %s: %+v\n", v.HexDigest(), spew.Sdump(v))
	fmt.Printf("Bytes:\n%s\n", hex.EncodeToString(resp.VaaBytes))
}

func runSendObservationRequest(cmd *cobra.Command, args []string) {
	chainID, err := parseChainID(args[0])
	if err != nil {
		log.Fatalf("invalid chain ID: %v", err)
	}

	txHash, err := hex.DecodeString(args[1])
	if err != nil {
		txHash, err = base58.Decode(args[1])
		if err != nil {
			log.Fatalf("invalid transaction hash (neither hex nor base58): %v", err)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, c, err := getAdminClient(ctx, *clientSocketPath)
	if err != nil {
		log.Fatalf("failed to get admin client: %v", err)
	}
	defer conn.Close()

	_, err = c.SendObservationRequest(ctx, &nodev1.SendObservationRequestRequest{
		ObservationRequest: &gossipv1.ObservationRequest{
			ChainId: uint32(chainID),
			TxHash:  txHash,
		},
	})
	if err != nil {
		log.Fatalf("failed to send observation request: %v", err)
	}
}

func runChainGovernorStatus(cmd *cobra.Command, args []string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, c, err := getAdminClient(ctx, *clientSocketPath)
	if err != nil {
		log.Fatalf("failed to get admin client: %v", err)
	}
	defer conn.Close()

	msg := nodev1.ChainGovernorStatusRequest{}
	resp, err := c.ChainGovernorStatus(ctx, &msg)
	if err != nil {
		log.Fatalf("failed to run ChainGovernorStatus RPC: %s", err)
	}

	fmt.Println(resp.Response)
}

func runChainGovernorReload(cmd *cobra.Command, args []string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, c, err := getAdminClient(ctx, *clientSocketPath)
	if err != nil {
		log.Fatalf("failed to get admin client: %v", err)
	}
	defer conn.Close()

	msg := nodev1.ChainGovernorReloadRequest{}
	resp, err := c.ChainGovernorReload(ctx, &msg)
	if err != nil {
		log.Fatalf("failed to run ChainGovernorReload RPC: %s", err)
	}

	fmt.Println(resp.Response)
}

func runChainGovernorDropPendingVAA(cmd *cobra.Command, args []string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, c, err := getAdminClient(ctx, *clientSocketPath)
	if err != nil {
		log.Fatalf("failed to get admin client: %v", err)
	}
	defer conn.Close()

	msg := nodev1.ChainGovernorDropPendingVAARequest{
		VaaId: args[0],
	}
	resp, err := c.ChainGovernorDropPendingVAA(ctx, &msg)
	if err != nil {
		log.Fatalf("failed to run ChainGovernorDropPendingVAA RPC: %s", err)
	}

	fmt.Println(resp.Response)
}

func runChainGovernorReleasePendingVAA(cmd *cobra.Command, args []string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, c, err := getAdminClient(ctx, *clientSocketPath)
	if err != nil {
		log.Fatalf("failed to get admin client: %v", err)
	}
	defer conn.Close()

	msg := nodev1.ChainGovernorReleasePendingVAARequest{
		VaaId: args[0],
	}
	resp, err := c.ChainGovernorReleasePendingVAA(ctx, &msg)
	if err != nil {
		log.Fatalf("failed to run ChainGovernorReleasePendingVAA RPC: %s", err)
	}

	fmt.Println(resp.Response)
}
