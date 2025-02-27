package guardiand

import (
	"bufio"
	"context"
	"encoding/csv"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/cosmos/cosmos-sdk/types"
	"github.com/davecgh/go-spew/spew"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/mr-tron/base58"
	"github.com/spf13/pflag"
	"golang.org/x/crypto/sha3"

	"github.com/certusone/wormhole/node/pkg/guardiansigner"
	"github.com/certusone/wormhole/node/pkg/node"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	publicrpcv1 "github.com/certusone/wormhole/node/pkg/proto/publicrpc/v1"
	"github.com/wormhole-foundation/wormhole/sdk"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"

	"github.com/spf13/cobra"
	"github.com/status-im/keycard-go/hexutils"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/encoding/prototext"

	nodev1 "github.com/certusone/wormhole/node/pkg/proto/node/v1"
)

var (
	clientSocketPath *string
	shouldBackfill   *bool
	unsafeDevnetMode *bool
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
	DumpRPCs.Flags().AddFlagSet(pf)
	SendObservationRequest.Flags().AddFlagSet(pf)
	ReobserveWithEndpoint.Flags().AddFlagSet(pf)
	ClientChainGovernorStatusCmd.Flags().AddFlagSet(pf)
	ClientChainGovernorReloadCmd.Flags().AddFlagSet(pf)
	ClientChainGovernorDropPendingVAACmd.Flags().AddFlagSet(pf)
	ClientChainGovernorReleasePendingVAACmd.Flags().AddFlagSet(pf)
	ClientChainGovernorResetReleaseTimerCmd.Flags().AddFlagSet(pf)
	PurgePythNetVaasCmd.Flags().AddFlagSet(pf)
	SignExistingVaaCmd.Flags().AddFlagSet(pf)
	SignExistingVaasFromCSVCmd.Flags().AddFlagSet(pf)
	GetAndObserveMissingVAAs.Flags().AddFlagSet(pf)

	adminClientSignWormchainAddressFlags := pflag.NewFlagSet("adminClientSignWormchainAddressFlags", pflag.ContinueOnError)
	unsafeDevnetMode = adminClientSignWormchainAddressFlags.Bool("unsafeDevMode", false, "Run in unsafe devnet mode")
	AdminClientSignWormchainAddress.Flags().AddFlagSet(adminClientSignWormchainAddressFlags)

	AdminCmd.AddCommand(AdminClientInjectGuardianSetUpdateCmd)
	AdminCmd.AddCommand(AdminClientFindMissingMessagesCmd)
	AdminCmd.AddCommand(AdminClientGovernanceVAAVerifyCmd)
	AdminCmd.AddCommand(AdminClientListNodes)
	AdminCmd.AddCommand(AdminClientSignWormchainAddress)
	AdminCmd.AddCommand(DumpVAAByMessageID)
	AdminCmd.AddCommand(DumpRPCs)
	AdminCmd.AddCommand(SendObservationRequest)
	AdminCmd.AddCommand(ReobserveWithEndpoint)
	AdminCmd.AddCommand(ClientChainGovernorStatusCmd)
	AdminCmd.AddCommand(ClientChainGovernorReloadCmd)
	AdminCmd.AddCommand(ClientChainGovernorDropPendingVAACmd)
	AdminCmd.AddCommand(ClientChainGovernorReleasePendingVAACmd)
	AdminCmd.AddCommand(ClientChainGovernorResetReleaseTimerCmd)
	AdminCmd.AddCommand(PurgePythNetVaasCmd)
	AdminCmd.AddCommand(SignExistingVaaCmd)
	AdminCmd.AddCommand(SignExistingVaasFromCSVCmd)
	AdminCmd.AddCommand(Keccak256Hash)
	AdminCmd.AddCommand(GetAndObserveMissingVAAs)
}

var AdminCmd = &cobra.Command{
	Use:   "admin",
	Short: "Guardian node admin commands",
}

var AdminClientSignWormchainAddress = &cobra.Command{
	Use:   "sign-wormchain-address [vaa-signer-uri] [wormchain-validator-address]",
	Short: "Sign a wormchain validator address.  Only sign the address that you control the key for and will be for your validator.",
	RunE:  runSignWormchainValidatorAddress,
	Args:  cobra.ExactArgs(2),
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

var ReobserveWithEndpoint = &cobra.Command{
	Use:   "reobserve-with-endpoint [CHAIN_ID|CHAIN_NAME] [TX_HASH_HEX] [CUSTOM_URL]",
	Short: "Performs a local reobservation for the given chain ID and chain-specific tx_hash using the specified endpoint",
	Run:   runReobserveWithEndpoint,
	Args:  cobra.ExactArgs(3),
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

var ClientChainGovernorResetReleaseTimerCmd = &cobra.Command{
	Use:   "governor-reset-release-timer [VAA_ID] <num_days>",
	Short: "Resets the release timer for a chain governor pending VAA, extending it to num_days (up to a maximum of 7), defaulting to one day if num_days is omitted",
	Run:   runChainGovernorResetReleaseTimer,
	Args:  cobra.RangeArgs(1, 2),
}

var PurgePythNetVaasCmd = &cobra.Command{
	Use:   "purge-pythnet-vaas [DAYS_OLD] <logonly>",
	Short: "Deletes PythNet VAAs from the database that are more than [DAYS_OLD] days only (if logonly is specified, doesn't delete anything)",
	Run:   runPurgePythNetVaas,
	Args:  cobra.RangeArgs(1, 2),
}

var SignExistingVaaCmd = &cobra.Command{
	Use:   "sign-existing-vaa [VAA] [NEW_GUARDIANS] [NEW_GUARDIAN_SET_INDEX]",
	Short: "Signs an existing VAA for a new guardian set using the local guardian key. This only works if the new VAA would have quorum.",
	Run:   runSignExistingVaa,
	Args:  cobra.ExactArgs(3),
}

var SignExistingVaasFromCSVCmd = &cobra.Command{
	Use:   "sign-existing-vaas-csv [IN_FILE] [OUT_FILE] [NEW_GUARDIANS] [NEW_GUARDIAN_SET_INDEX]",
	Short: "Signs a CSV [VAA_ID,VAA_HEX] of existing VAAs for a new guardian set using the local guardian key and writes it to a new CSV. VAAs that don't have quorum on the new set will be dropped.",
	Run:   runSignExistingVaasFromCSV,
	Args:  cobra.ExactArgs(4),
}

var DumpRPCs = &cobra.Command{
	Use:   "dump-rpcs",
	Short: "Displays the RPCs in use by the guardian",
	Run:   runDumpRPCs,
	Args:  cobra.ExactArgs(0),
}

var GetAndObserveMissingVAAs = &cobra.Command{
	Use:   "get-and-observe-missing-vaas [URL] [API_KEY]",
	Short: "Get the list of missing VAAs from a cloud function and try to reobserve them.",
	Run:   runGetAndObserveMissingVAAs,
	Args:  cobra.ExactArgs(2),
}

var Keccak256Hash = &cobra.Command{
	Use:   "keccak256",
	Short: "Compute legacy keccak256 hash",
	Run:   runKeccak256Hash,
	Args:  cobra.ExactArgs(0),
}

func getAdminClient(ctx context.Context, addr string) (*grpc.ClientConn, nodev1.NodePrivilegedServiceClient, error) {
	conn, err := grpc.DialContext(ctx, fmt.Sprintf("unix:///%s", addr), grpc.WithTransportCredentials(insecure.NewCredentials()))

	if err != nil {
		log.Fatalf("failed to connect to %s: %v", addr, err)
	}

	c := nodev1.NewNodePrivilegedServiceClient(conn)
	return conn, c, err
}

func getPublicRPCServiceClient(ctx context.Context, addr string) (*grpc.ClientConn, publicrpcv1.PublicRPCServiceClient, error) {
	conn, err := grpc.DialContext(ctx, fmt.Sprintf("unix:///%s", addr), grpc.WithTransportCredentials(insecure.NewCredentials()))

	if err != nil {
		log.Fatalf("failed to connect to %s: %v", addr, err)
	}

	c := publicrpcv1.NewPublicRPCServiceClient(conn)
	return conn, c, err
}

func runSignWormchainValidatorAddress(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	guardianSignerUri := args[0]
	wormchainAddress := args[1]
	if !strings.HasPrefix(wormchainAddress, "wormhole") || strings.HasPrefix(wormchainAddress, "wormholeval") {
		return errors.New("must provide a bech32 address that has 'wormhole' prefix")
	}

	guardianSigner, err := guardiansigner.NewGuardianSignerFromUri(ctx, guardianSignerUri, *unsafeDevnetMode)
	if err != nil {
		return fmt.Errorf("failed to create new guardian signer from uri: %w", err)
	}

	addr, err := types.GetFromBech32(wormchainAddress, "wormhole")
	if err != nil {
		return fmt.Errorf("failed to decode wormchain address: %w", err)
	}

	// Hash and sign address
	addrHash := crypto.Keccak256Hash(sdk.SignedWormchainAddressPrefix, addr)
	sig, err := guardianSigner.Sign(ctx, addrHash.Bytes())
	if err != nil {
		return fmt.Errorf("failed to sign wormchain address: %w", err)
	}
	fmt.Println(hex.EncodeToString(sig))
	return nil
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

	b, err := os.ReadFile(path)
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

	if chainID > math.MaxUint16 {
		log.Fatalf("chain ID is not a valid 16 bit unsigned integer: %v", err)
	}

	msg := nodev1.FindMissingMessagesRequest{
		EmitterChain:   uint32(chainID), // #nosec G115 -- This conversion is checked above
		EmitterAddress: emitterAddress,
		RpcBackfill:    *shouldBackfill,
		BackfillNodes:  sdk.PublicRPCEndpoints,
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
	if chainID > math.MaxUint16 {
		log.Fatalf("chain id must not exceed the max uint16: %v", chainID)
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

	// Support tx with or without leading 0x so copy / pasta
	// from monitoring tools is easier.
	txHash, err := hex.DecodeString(strings.TrimPrefix(args[1], "0x"))
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

func runReobserveWithEndpoint(cmd *cobra.Command, args []string) {
	chainID, err := parseChainID(args[0])
	if err != nil {
		log.Fatalf("invalid chain ID: %v", err)
	}

	// Support tx with or without leading 0x.
	txHash, err := hex.DecodeString(strings.TrimPrefix(args[1], "0x"))
	if err != nil {
		txHash, err = base58.Decode(args[1])
		if err != nil {
			log.Fatalf("invalid transaction hash (neither hex nor base58): %v", err)
		}
	}

	url := args[2]
	if valid := node.ValidateURL(url, []string{"http", "https"}); !valid {
		log.Fatalf(`invalid url, must be "http" or "https"`)
	}

	// Allow extra time since the watcher can block on the reobservation.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	conn, c, err := getAdminClient(ctx, *clientSocketPath)
	if err != nil {
		log.Fatalf("failed to get admin client: %v", err)
	}
	defer conn.Close()

	resp, err := c.ReobserveWithEndpoint(ctx, &nodev1.ReobserveWithEndpointRequest{
		ChainId: uint32(chainID),
		TxHash:  txHash,
		Url:     url,
	})
	if err != nil {
		log.Fatalf("failed to send observation request with endpoint: %v", err)
	}
	if resp.NumObservations == 0 {
		fmt.Println("Did not reobserve anything")
	} else {
		fmt.Println("Reobserved", resp.NumObservations, "messages")
	}
}

func runDumpRPCs(cmd *cobra.Command, args []string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, c, err := getAdminClient(ctx, *clientSocketPath)
	if err != nil {
		log.Fatalf("failed to get admin client: %v", err)
	}
	defer conn.Close()

	resp, err := c.DumpRPCs(ctx, &nodev1.DumpRPCsRequest{})
	if err != nil {
		log.Fatalf("failed to run dump-rpcs: %s", err)
	}

	for parm, rpc := range resp.Response {
		fmt.Println(parm, " = [", rpc, "]")
	}
}

func runGetAndObserveMissingVAAs(cmd *cobra.Command, args []string) {
	url := args[0]
	if !strings.HasPrefix(url, "https://") {
		log.Fatalf("invalid url: %s", url)
	}
	apiKey := args[1]
	if len(apiKey) == 0 {
		log.Fatalf("missing api key")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	conn, c, err := getAdminClient(ctx, *clientSocketPath)
	if err != nil {
		log.Fatalf("failed to get admin client: %v", err)
	}
	defer conn.Close()

	cmdInfo := nodev1.GetAndObserveMissingVAAsRequest{
		Url:    url,
		ApiKey: apiKey,
	}
	resp, err := c.GetAndObserveMissingVAAs(ctx, &cmdInfo)
	if err != nil {
		log.Fatalf("failed to run get-missing-vaas: %s", err)
	}

	fmt.Println(resp.GetResponse())
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

func runChainGovernorResetReleaseTimer(cmd *cobra.Command, args []string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, c, err := getAdminClient(ctx, *clientSocketPath)
	if err != nil {
		log.Fatalf("failed to get admin client: %v", err)
	}
	defer conn.Close()

	// defaults to 1 day if num_days isn't specified
	numDays := uint32(1)
	if len(args) > 1 {
		numDaysArg, err := strconv.Atoi(args[1])

		if numDaysArg > math.MaxUint32 || err != nil {
			log.Fatalf("invalid num_days: %v", err)
		}

		numDays = uint32(numDaysArg) // #nosec G115 -- This is validated above
	}

	msg := nodev1.ChainGovernorResetReleaseTimerRequest{
		VaaId:   args[0],
		NumDays: numDays,
	}
	resp, err := c.ChainGovernorResetReleaseTimer(ctx, &msg)
	if err != nil {
		log.Fatalf("failed to run ChainGovernorResetReleaseTimer RPC: %s", err)
	}

	fmt.Println(resp.Response)
}

func runPurgePythNetVaas(cmd *cobra.Command, args []string) {
	daysOld, err := strconv.Atoi(args[0])
	if err != nil {
		log.Fatalf("invalid DAYS_OLD: %v", err)
	}

	if daysOld < 0 {
		log.Fatalf("DAYS_OLD may not be negative")
	}

	logOnly := false
	if len(args) > 1 {
		if args[1] != "logonly" {
			log.Fatalf("invalid option, only \"logonly\" is supported")
		}

		logOnly = true
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conn, c, err := getAdminClient(ctx, *clientSocketPath)
	if err != nil {
		log.Fatalf("failed to get admin client: %v", err)
	}
	defer conn.Close()

	msg := nodev1.PurgePythNetVaasRequest{
		DaysOld: uint64(daysOld),
		LogOnly: logOnly,
	}
	resp, err := c.PurgePythNetVaas(ctx, &msg)
	if err != nil {
		log.Fatalf("failed to run PurgePythNetVaas RPC: %s", err)
	}

	fmt.Println(resp.Response)
}

func runSignExistingVaa(cmd *cobra.Command, args []string) {
	existingVAA := ethcommon.Hex2Bytes(args[0])
	if len(existingVAA) == 0 {
		log.Fatalf("vaa hex invalid")
	}

	newGsStrings := strings.Split(args[1], ",")

	newGsIndex, err := strconv.ParseUint(args[2], 10, 32)
	if err != nil {
		log.Fatalf("invalid new guardian set index")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conn, c, err := getAdminClient(ctx, *clientSocketPath)
	if err != nil {
		log.Fatalf("failed to get admin client: %v", err)
	}
	defer conn.Close()

	msg := nodev1.SignExistingVAARequest{
		Vaa:                 existingVAA,
		NewGuardianAddrs:    newGsStrings,
		NewGuardianSetIndex: uint32(newGsIndex),
	}
	resp, err := c.SignExistingVAA(ctx, &msg)
	if err != nil {
		log.Fatalf("failed to run SignExistingVAA RPC: %s", err)
	}

	fmt.Println(hex.EncodeToString(resp.Vaa))
}

func runSignExistingVaasFromCSV(cmd *cobra.Command, args []string) {
	oldVAAFile, err := os.Open(args[0])
	if err != nil {
		log.Fatalf("failed to read old VAA db: %v", err)
	}
	defer oldVAAFile.Close()

	newVAAFile, err := os.OpenFile(args[1], os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		log.Fatalf("failed to create new VAA db: %v", err)
	}
	defer newVAAFile.Close()
	newVAAWriter := csv.NewWriter(newVAAFile)

	newGsStrings := strings.Split(args[2], ",")

	newGsIndex, err := strconv.ParseUint(args[3], 10, 32)
	if err != nil {
		log.Fatalf("invalid new guardian set index")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	conn, c, err := getAdminClient(ctx, *clientSocketPath)
	if err != nil {
		log.Fatalf("failed to get admin client: %v", err)
	}
	defer conn.Close()

	// Scan the CSV once to make sure it won't fail while reading unless raced
	oldVAAReader := csv.NewReader(oldVAAFile)
	numOldVAAs := 0
	for {
		row, err := oldVAAReader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Fatalf("failed to parse VAA CSV: %v", err)
		}
		if len(row) != 2 {
			log.Fatalf("row [%d] does not have 2 elements", numOldVAAs)
		}
		numOldVAAs++
	}

	// Reset reader
	_, err = oldVAAFile.Seek(0, io.SeekStart)
	if err != nil {
		log.Fatalf("failed to seek back in CSV file: %v", err)
	}
	oldVAAReader = csv.NewReader(oldVAAFile)

	counter, i := 0, 0
	for {
		row, err := oldVAAReader.Read()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			log.Fatalf("failed to parse VAA CSV: %v", err)
		}
		if len(row) != 2 {
			log.Fatalf("row [%d] does not have 2 elements", i)
		}
		i++

		if i%10 == 0 {
			log.Printf("Processing VAA %d/%d", i, numOldVAAs)
		}

		vaaBytes := ethcommon.Hex2Bytes(row[1])
		msg := nodev1.SignExistingVAARequest{
			Vaa:                 vaaBytes,
			NewGuardianAddrs:    newGsStrings,
			NewGuardianSetIndex: uint32(newGsIndex),
		}
		resp, err := c.SignExistingVAA(ctx, &msg)
		if err != nil {
			log.Printf("signing VAA (%s)[%d] failed - skipping: %v", row[0], i, err)
			continue
		}
		err = newVAAWriter.Write([]string{row[0], hex.EncodeToString(resp.Vaa)})
		if err != nil {
			log.Fatalf("failed to write new VAA to out db: %v", err)
		}
		counter++
	}

	log.Printf("Successfully signed %d out of %d VAAs", counter, numOldVAAs)
	newVAAWriter.Flush()
}

// This exposes keccak256 as a command line utility, mostly for validating governance messages
// that use this hash.  There isn't any common utility that computes this since this is nonstandard outside of evm.
// It is used similar to other hashing utilities, e.g. `cat <file> | guardiand admin keccak256`.
func runKeccak256Hash(cmd *cobra.Command, args []string) {
	reader := bufio.NewReader(os.Stdin)
	hash := sha3.NewLegacyKeccak256()
	// ~10 MB chunks
	buf := make([]byte, 10*1024*1024)
	for {
		count, err := reader.Read(buf)
		if err != nil && err != io.EOF {
			log.Fatalf("could not read: %v", err)
		}
		_, errHash := hash.Write(buf[:count])
		if errHash != nil {
			log.Fatalf("could not hash: %v", errHash)
		}
		if err == io.EOF {
			break
		}
	}
	digest := hash.Sum([]byte{})
	fmt.Printf("%s", hex.EncodeToString(digest))
}
