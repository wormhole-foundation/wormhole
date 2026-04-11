package guardiand

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/cosmos/cosmos-sdk/types"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/mr-tron/base58"
	"github.com/spf13/pflag"
	"golang.org/x/crypto/sha3"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/guardiansigner"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	publicrpcv1 "github.com/certusone/wormhole/node/pkg/proto/publicrpc/v1"
	"github.com/wormhole-foundation/wormhole/sdk"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"

	"github.com/spf13/cobra"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/encoding/prototext"

	nodev1 "github.com/certusone/wormhole/node/pkg/proto/node/v1"

	"google.golang.org/protobuf/proto"
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
	NotaryBlackholeDelayedMessage.Flags().AddFlagSet(pf)
	NotaryReleaseDelayedMessage.Flags().AddFlagSet(pf)
	NotaryRemoveBlackholedMessage.Flags().AddFlagSet(pf)
	NotaryResetReleaseTimer.Flags().AddFlagSet(pf)
	NotaryInjectDelayedMessage.Flags().AddFlagSet(pf)
	NotaryInjectBlackholedMessage.Flags().AddFlagSet(pf)
	NotaryGetDelayedMessage.Flags().AddFlagSet(pf)
	NotaryGetBlackholedMessage.Flags().AddFlagSet(pf)
	NotaryListDelayedMessages.Flags().AddFlagSet(pf)
	NotaryListBlackholedMessages.Flags().AddFlagSet(pf)
	PurgePythNetVaasCmd.Flags().AddFlagSet(pf)
	SignExistingVaaCmd.Flags().AddFlagSet(pf)
	SignExistingVaasFromCSVCmd.Flags().AddFlagSet(pf)
	GetAndObserveMissingVAAs.Flags().AddFlagSet(pf)
	BroadcastDelegateSignatures.Flags().AddFlagSet(pf)

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
	// Chain governor commands
	AdminCmd.AddCommand(ClientChainGovernorStatusCmd)
	AdminCmd.AddCommand(ClientChainGovernorReloadCmd)
	AdminCmd.AddCommand(ClientChainGovernorDropPendingVAACmd)
	AdminCmd.AddCommand(ClientChainGovernorReleasePendingVAACmd)
	AdminCmd.AddCommand(ClientChainGovernorResetReleaseTimerCmd)
	// Notary commands
	AdminCmd.AddCommand(NotaryBlackholeDelayedMessage)
	AdminCmd.AddCommand(NotaryReleaseDelayedMessage)
	AdminCmd.AddCommand(NotaryRemoveBlackholedMessage)
	AdminCmd.AddCommand(NotaryResetReleaseTimer)
	AdminCmd.AddCommand(NotaryInjectDelayedMessage)
	AdminCmd.AddCommand(NotaryInjectBlackholedMessage)
	AdminCmd.AddCommand(NotaryGetDelayedMessage)
	AdminCmd.AddCommand(NotaryGetBlackholedMessage)
	AdminCmd.AddCommand(NotaryListDelayedMessages)
	AdminCmd.AddCommand(NotaryListBlackholedMessages)
	// Other commands
	AdminCmd.AddCommand(PurgePythNetVaasCmd)
	AdminCmd.AddCommand(SignExistingVaaCmd)
	AdminCmd.AddCommand(SignExistingVaasFromCSVCmd)
	AdminCmd.AddCommand(Keccak256Hash)
	AdminCmd.AddCommand(GetAndObserveMissingVAAs)
	AdminCmd.AddCommand(BroadcastDelegateSignatures)
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
	Short: "Resets the release timer for a chain governor pending VAA, extending it to num_days (up to a maximum of 30), defaulting to one day if num_days is omitted",
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

var BroadcastDelegateSignatures = &cobra.Command{
	Use:   "broadcast-delegate-signatures [VAA_ID]",
	Short: "Fetch delegate signatures from wormholescan and broadcast them on the delegated attestation topic",
	Run:   runBroadcastDelegateSignatures,
	Args:  cobra.ExactArgs(1),
}

var Keccak256Hash = &cobra.Command{
	Use:   "keccak256",
	Short: "Compute legacy keccak256 hash",
	Run:   runKeccak256Hash,
	Args:  cobra.ExactArgs(0),
}

// Notary commands
var (
	NotaryBlackholeDelayedMessage = &cobra.Command{
		Use:   "notary-blackhole-delayed-message [MESSAGE_ID]",
		Short: "Moves the specified VAA (chain/emitter/seq) from the Notary's delayed list to the blackholed list",
		Run:   runNotaryBlackholeDelayedMessage,
		Args:  cobra.ExactArgs(1),
	}

	NotaryReleaseDelayedMessage = &cobra.Command{
		Use:   "notary-release-delayed-message [MESSAGE_ID]",
		Short: "Releases the specified VAA (chain/emitter/seq) from the Notary's delayed list, publishing it immediately",
		Run:   runNotaryReleaseDelayedMessage,
		Args:  cobra.ExactArgs(1),
	}

	NotaryRemoveBlackholedMessage = &cobra.Command{
		Use:   "notary-remove-blackholed-message [MESSAGE_ID]",
		Short: "Removes the specified VAA (chain/emitter/seq) from the Notary's blackholed list and adds it to the delayed list with a delay of zero",
		Run:   runNotaryRemoveBlackholedMessage,
		Args:  cobra.ExactArgs(1),
	}
	NotaryResetReleaseTimer = &cobra.Command{
		Use:   "notary-reset-release-timer [MESSAGE_ID] [DELAY_DAYS]",
		Short: "Resets the release timer for a notary pending VAA to supplied number of days.",
		Run:   runNotaryResetReleaseTimer,
		Args:  cobra.ExactArgs(2),
	}

	NotaryInjectDelayedMessage = &cobra.Command{
		Use:   "notary-inject-delayed-message [DELAY_DAYS]",
		Short: "Injects a synthetic delayed message for testing (dev mode only)",
		Run:   runNotaryInjectDelayedMessage,
		Args:  cobra.ExactArgs(1),
	}

	NotaryInjectBlackholedMessage = &cobra.Command{
		Use:   "notary-inject-blackholed-message",
		Short: "Injects a synthetic blackholed message for testing (dev mode only)",
		Run:   runNotaryInjectBlackholedMessage,
		Args:  cobra.NoArgs,
	}

	NotaryGetDelayedMessage = &cobra.Command{
		Use:   "notary-get-delayed-message [MESSAGE_ID]",
		Short: "Gets details about a delayed message",
		Run:   runNotaryGetDelayedMessage,
		Args:  cobra.ExactArgs(1),
	}

	NotaryGetBlackholedMessage = &cobra.Command{
		Use:   "notary-get-blackholed-message [MESSAGE_ID]",
		Short: "Gets details about a blackholed message",
		Run:   runNotaryGetBlackholedMessage,
		Args:  cobra.ExactArgs(1),
	}

	NotaryListDelayedMessages = &cobra.Command{
		Use:   "notary-list-delayed-messages",
		Short: "Lists all delayed message IDs",
		Run:   runNotaryListDelayedMessages,
		Args:  cobra.NoArgs,
	}

	NotaryListBlackholedMessages = &cobra.Command{
		Use:   "notary-list-blackholed-messages",
		Short: "Lists all blackholed message IDs",
		Run:   runNotaryListBlackholedMessages,
		Args:  cobra.NoArgs,
	}
)

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
		log.Printf("VAA successfully injected with digest %x", digest)
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

	debugStr, err := v.DebugString()
	if err != nil {
		log.Fatal(err.Error())
	}
	log.Printf("VAA with digest %s: %+v\n", v.HexDigest(), debugStr)
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
			ChainId:   uint32(chainID),
			TxHash:    txHash,
			Timestamp: time.Now().UnixNano(),
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
	if valid := common.ValidateURL(url, []string{"http", "https"}); !valid {
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

// wormholescanDelegateObservation represents a delegate observation from the wormholescan API.
type wormholescanDelegateObservation struct {
	Sequence               uint64 `json:"sequence"`
	EmitterChain           uint32 `json:"emitterChain"`
	EmitterAddr            string `json:"emitterAddr"`
	MessagePublicationHash string `json:"hash"`
	TxHash                 string `json:"txHash"`
	Payload                string `json:"payload"`
	DelegatedGuardianAddr  string `json:"delegatedGuardianAddr"`
	Signature              string `json:"signature"`
	Nonce                  uint32 `json:"nonce"`
	ConsistencyLevel       uint32 `json:"consistencyLevel"`
	Timestamp              string `json:"timestamp"`
	SentTimestamp          string `json:"sentTimestamp"`
	Unreliable             bool   `json:"unreliable"`
	IsReobservation        bool   `json:"isReobservation"`
	VerificationState      uint32 `json:"verificationState"`
}

// delegateObservationVAAHash computes the VAA body hash (double Keccak256) from a delegate observation's fields.
func delegateObservationVAAHash(obs *wormholescanDelegateObservation) (string, error) {
	ts, err := time.Parse(time.RFC3339, obs.Timestamp)
	if err != nil {
		return "", fmt.Errorf("failed to parse timestamp: %v", err)
	}
	payload, err := base64.StdEncoding.DecodeString(obs.Payload)
	if err != nil {
		return "", fmt.Errorf("failed to decode payload: %v", err)
	}
	emitterAddr, err := hex.DecodeString(obs.EmitterAddr)
	if err != nil {
		return "", fmt.Errorf("failed to decode emitter address: %v", err)
	}
	// VAA body: timestamp(4) || nonce(4) || emitter_chain(2) || emitter_address(32) || sequence(8) || consistency_level(1) || payload
	buf := make([]byte, 0, 4+4+2+32+8+1+len(payload))
	b := make([]byte, 8)
	binary.BigEndian.PutUint32(b[:4], uint32(ts.Unix())) // #nosec G115 -- timestamp fits in uint32
	buf = append(buf, b[:4]...)
	binary.BigEndian.PutUint32(b[:4], obs.Nonce)
	buf = append(buf, b[:4]...)
	binary.BigEndian.PutUint16(b[:2], uint16(obs.EmitterChain)) // #nosec G115 -- chain ID fits in uint16
	buf = append(buf, b[:2]...)
	buf = append(buf, emitterAddr...)
	binary.BigEndian.PutUint64(b, obs.Sequence)
	buf = append(buf, b...)
	buf = append(buf, uint8(obs.ConsistencyLevel)) // #nosec G115 -- consistency level fits in uint8
	buf = append(buf, payload...)
	return crypto.Keccak256Hash(crypto.Keccak256Hash(buf).Bytes()).Hex(), nil
}

// buildDelegateSignaturesBroadcasts takes parsed wormholescan API observations and a VAA ID,
// groups by VAA body hash to find the best set of signatures, then sub-groups by
// MessagePublicationHash (which includes non-VAA fields like TxHash) to produce one
// broadcast per unique message publication. This is necessary because a DelegateSignaturesBroadcast
// carries common fields (including TxHash) that must be identical for all signatures in a batch.
func buildDelegateSignaturesBroadcasts(vaaID string, apiObservations []wormholescanDelegateObservation) ([]*gossipv1.DelegateSignaturesBroadcast, error) {
	parts := strings.Split(vaaID, "/")
	if len(parts) != 3 {
		return nil, fmt.Errorf("vaa_id must be in format chain/emitter/sequence")
	}
	chainID := parts[0]

	if len(apiObservations) == 0 {
		return nil, fmt.Errorf("no delegate observations provided")
	}

	// Step 1: Group observations by VAA body hash — pick the group with the most signatures.
	type vaaGroup struct {
		observations []wormholescanDelegateObservation
	}
	vaaGroups := make(map[string]*vaaGroup)
	for i := range apiObservations {
		h, err := delegateObservationVAAHash(&apiObservations[i])
		if err != nil {
			continue
		}
		g, ok := vaaGroups[h]
		if !ok {
			g = &vaaGroup{}
			vaaGroups[h] = g
		}
		g.observations = append(g.observations, apiObservations[i])
	}

	var bestVAAGroup *vaaGroup
	for _, g := range vaaGroups {
		if bestVAAGroup == nil || len(g.observations) > len(bestVAAGroup.observations) {
			bestVAAGroup = g
		}
	}

	if bestVAAGroup == nil || len(bestVAAGroup.observations) == 0 {
		return nil, fmt.Errorf("no valid observation groups found")
	}

	// Step 2: Sub-group by MessagePublicationHash so each broadcast has consistent common fields.
	type mpGroup struct {
		observations []wormholescanDelegateObservation
	}
	mpGroups := make(map[string]*mpGroup)
	for _, obs := range bestVAAGroup.observations {
		h := obs.MessagePublicationHash
		if h == "" {
			continue
		}
		g, ok := mpGroups[h]
		if !ok {
			g = &mpGroup{}
			mpGroups[h] = g
		}
		g.observations = append(g.observations, obs)
	}

	chainIDParsed, err := vaa.StringToKnownChainID(chainID)
	if err != nil {
		return nil, fmt.Errorf("invalid chain ID: %v", err)
	}
	chainIDNum := int(chainIDParsed)

	// Step 3: Build one broadcast per MessagePublicationHash sub-group.
	signedDelegateObservationPrefix := []byte("signed_delegate_observation_000000|")
	broadcasts := make([]*gossipv1.DelegateSignaturesBroadcast, 0, len(mpGroups))

	for _, mg := range mpGroups {
		ref := mg.observations[0]

		emitterAddrBytes, err := hex.DecodeString(ref.EmitterAddr)
		if err != nil {
			log.Printf("Warning: skipping group with invalid emitter address: %v", err)
			continue
		}

		txHashBytes, err := base64.StdEncoding.DecodeString(ref.TxHash)
		if err != nil {
			log.Printf("Warning: skipping group with invalid tx hash: %v", err)
			continue
		}

		payloadBytes, err := base64.StdEncoding.DecodeString(ref.Payload)
		if err != nil {
			log.Printf("Warning: skipping group with invalid payload: %v", err)
			continue
		}

		var timestamp uint32
		if ref.Timestamp != "" {
			t, err := time.Parse(time.RFC3339, ref.Timestamp)
			if err != nil {
				log.Printf("Warning: skipping group with invalid timestamp: %v", err)
				continue
			}
			timestamp = uint32(t.Unix()) // #nosec G115
		}

		signatures := make([]*gossipv1.DelegateSignature, 0, len(mg.observations))
		for _, obs := range mg.observations {
			guardianAddrHex := strings.TrimPrefix(obs.DelegatedGuardianAddr, "0x")
			guardianAddrBytes, err := hex.DecodeString(guardianAddrHex)
			if err != nil {
				log.Printf("Warning: skipping observation with invalid guardian address %q: %v", obs.DelegatedGuardianAddr, err)
				continue
			}

			var sigBytes []byte
			if obs.Signature != "" {
				sigBytes, err = base64.StdEncoding.DecodeString(obs.Signature)
				if err != nil {
					log.Printf("Warning: skipping observation with invalid signature for guardian %s: %v", obs.DelegatedGuardianAddr, err)
					continue
				}
			}

			var sentTimestamp int64
			if obs.SentTimestamp != "" {
				t, err := time.Parse(time.RFC3339, obs.SentTimestamp)
				if err != nil {
					continue
				}
				sentTimestamp = t.Unix()
			}

			// Reconstruct the DelegateObservation that was originally signed.
			d := &gossipv1.DelegateObservation{
				Timestamp:         timestamp,
				Nonce:             ref.Nonce,
				EmitterChain:      uint32(chainIDNum), // #nosec G115
				EmitterAddress:    emitterAddrBytes,
				Sequence:          ref.Sequence,
				ConsistencyLevel:  ref.ConsistencyLevel,
				Payload:           payloadBytes,
				TxHash:            txHashBytes,
				Unreliable:        obs.Unreliable,
				IsReobservation:   obs.IsReobservation,
				VerificationState: obs.VerificationState,
				GuardianAddr:      guardianAddrBytes,
				SentTimestamp:     sentTimestamp,
			}
			b, err := proto.Marshal(d)
			if err != nil {
				continue
			}

			// Verify the signature over the reconstructed bytes.
			if len(sigBytes) == 0 {
				continue
			}
			digest := crypto.Keccak256Hash(append(signedDelegateObservationPrefix, b...))
			pubKey, err := crypto.Ecrecover(digest.Bytes(), sigBytes)
			if err != nil {
				continue
			}
			signerAddr := ethcommon.BytesToAddress(crypto.Keccak256(pubKey[1:])[12:])
			claimedAddr := ethcommon.BytesToAddress(guardianAddrBytes)
			if signerAddr != claimedAddr {
				continue
			}

			signatures = append(signatures, &gossipv1.DelegateSignature{
				GuardianAddr:  guardianAddrBytes,
				SentTimestamp: sentTimestamp,
				Signature:     sigBytes,
			})
		}

		if len(signatures) == 0 {
			continue
		}

		broadcasts = append(broadcasts, &gossipv1.DelegateSignaturesBroadcast{
			Timestamp:          timestamp,
			Nonce:              ref.Nonce,
			EmitterChain:       uint32(chainIDNum), // #nosec G115
			EmitterAddress:     emitterAddrBytes,
			Sequence:           ref.Sequence,
			ConsistencyLevel:   ref.ConsistencyLevel,
			Payload:            payloadBytes,
			TxHash:             txHashBytes,
			Unreliable:         ref.Unreliable,
			IsReobservation:    ref.IsReobservation,
			VerificationState:  ref.VerificationState,
			Signatures:         signatures,
			BroadcastTimestamp: time.Now().Unix(),
		})
	}

	if len(broadcasts) == 0 {
		return nil, fmt.Errorf("no signatures passed verification")
	}

	return broadcasts, nil
}

func runBroadcastDelegateSignatures(cmd *cobra.Command, args []string) {
	vaaID := args[0]

	// Parse VAA ID: chain/emitter/sequence
	parts := strings.Split(vaaID, "/")
	if len(parts) != 3 {
		log.Fatalf("vaa_id must be in format chain/emitter/sequence")
	}
	chainID := parts[0]
	emitterAddr := parts[1]
	sequence := parts[2]

	if _, err := vaa.StringToKnownChainID(chainID); err != nil {
		log.Fatalf("invalid chain ID %q: %v", chainID, err)
	}

	// Fetch delegate observations from wormholescan API.
	apiURL := fmt.Sprintf("https://api.wormholescan.io/api/v1/observations/delegate/%s/%s/%s", chainID, emitterAddr, sequence)
	httpReq, err := http.NewRequestWithContext(context.Background(), http.MethodGet, apiURL, nil)
	if err != nil {
		log.Fatalf("failed to create HTTP request: %v", err)
	}
	httpResp, err := (&http.Client{Timeout: 30 * time.Second}).Do(httpReq)
	if err != nil {
		log.Fatalf("failed to fetch delegate observations: %v", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		log.Fatalf("wormholescan API returned status %d", httpResp.StatusCode)
	}

	var apiObservations []wormholescanDelegateObservation
	if err := json.NewDecoder(httpResp.Body).Decode(&apiObservations); err != nil {
		log.Fatalf("failed to decode API response: %v", err)
	}

	broadcasts, err := buildDelegateSignaturesBroadcasts(vaaID, apiObservations)
	if err != nil {
		log.Fatalf("failed to build broadcasts: %v", err)
	}

	// Send to the guardian node for signing and p2p broadcast.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	conn, c, err := getAdminClient(ctx, *clientSocketPath)
	if err != nil {
		log.Fatalf("failed to get admin client: %v", err)
	}
	defer conn.Close()

	for i, broadcast := range broadcasts {
		fmt.Printf("broadcasting batch %d/%d with %d verified delegate signatures for %s\n",
			i+1, len(broadcasts), len(broadcast.Signatures), vaaID)

		resp, err := c.BroadcastDelegateSignatures(ctx, &nodev1.BroadcastDelegateSignaturesRequest{
			Broadcast: broadcast,
		})
		if err != nil {
			log.Fatalf("failed to broadcast delegate signatures (batch %d): %s", i+1, err)
		}

		fmt.Println(resp.GetResponse())
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

func runNotaryBlackholeDelayedMessage(cmd *cobra.Command, args []string) {
	ctx := context.Background()
	conn, c, err := getAdminClient(ctx, *clientSocketPath)
	if err != nil {
		log.Fatalf("failed to get admin client: %v", err)
	}
	defer conn.Close()

	msg := nodev1.NotaryBlackholeDelayedMessageRequest{
		VaaId: args[0],
	}
	resp, err := c.NotaryBlackholeDelayedMessage(ctx, &msg)
	if err != nil {
		log.Fatalf("failed to run NotaryBlackholeDelayedMessage RPC: %s", err)
	}

	fmt.Println(resp.Response)
}

func runNotaryReleaseDelayedMessage(cmd *cobra.Command, args []string) {
	ctx := context.Background()
	conn, c, err := getAdminClient(ctx, *clientSocketPath)
	if err != nil {
		log.Fatalf("failed to get admin client: %v", err)
	}
	defer conn.Close()

	msg := nodev1.NotaryReleaseDelayedMessageRequest{
		VaaId: args[0],
	}
	resp, err := c.NotaryReleaseDelayedMessage(ctx, &msg)
	if err != nil {
		log.Fatalf("failed to run NotaryReleaseDelayedMessage RPC: %s", err)
	}

	fmt.Println(resp.Response)
}

func runNotaryRemoveBlackholedMessage(cmd *cobra.Command, args []string) {
	ctx := context.Background()
	conn, c, err := getAdminClient(ctx, *clientSocketPath)
	if err != nil {
		log.Fatalf("failed to get admin client: %v", err)
	}
	defer conn.Close()

	msg := nodev1.NotaryRemoveBlackholedMessageRequest{
		VaaId: args[0],
	}
	resp, err := c.NotaryRemoveBlackholedMessage(ctx, &msg)
	if err != nil {
		log.Fatalf("failed to run NotaryRemoveBlackholedMessage RPC: %s", err)
	}

	fmt.Println(resp.Response)
}

func runNotaryResetReleaseTimer(cmd *cobra.Command, args []string) {
	ctx := context.Background()
	conn, c, err := getAdminClient(ctx, *clientSocketPath)
	if err != nil {
		log.Fatalf("failed to get admin client: %v", err)
	}
	defer conn.Close()

	delayDays, err := strconv.ParseUint(args[1], 10, 8)
	if err != nil {
		log.Fatalf("invalid delay days: %v", err)
	}

	msg := nodev1.NotaryResetReleaseTimerRequest{
		VaaId:   args[0],
		NumDays: uint32(delayDays), // protobuf does not support uint8
	}
	resp, err := c.NotaryResetReleaseTimer(ctx, &msg)
	if err != nil {
		log.Fatalf("failed to run NotaryResetReleaseTimer RPC: %s", err)
	}

	fmt.Println(resp.Response)
}

func runNotaryInjectDelayedMessage(cmd *cobra.Command, args []string) {
	ctx := context.Background()
	conn, c, err := getAdminClient(ctx, *clientSocketPath)
	if err != nil {
		log.Fatalf("failed to get admin client: %v", err)
	}
	defer conn.Close()

	delayDays, err := strconv.ParseUint(args[0], 10, 32)
	if err != nil {
		log.Fatalf("invalid delay days: %v", err)
	}

	msg := nodev1.NotaryInjectDelayedMessageRequest{
		DelayDays: uint32(delayDays),
	}
	resp, err := c.NotaryInjectDelayedMessage(ctx, &msg)
	if err != nil {
		log.Fatalf("failed to run NotaryInjectDelayedMessage RPC: %s", err)
	}

	fmt.Printf("Injected delayed message: %s\n", resp.VaaId)
}

func runNotaryInjectBlackholedMessage(cmd *cobra.Command, args []string) {
	ctx := context.Background()
	conn, c, err := getAdminClient(ctx, *clientSocketPath)
	if err != nil {
		log.Fatalf("failed to get admin client: %v", err)
	}
	defer conn.Close()

	msg := nodev1.NotaryInjectBlackholedMessageRequest{}
	resp, err := c.NotaryInjectBlackholedMessage(ctx, &msg)
	if err != nil {
		log.Fatalf("failed to run NotaryInjectBlackholedMessage RPC: %s", err)
	}

	fmt.Printf("Injected blackholed message: %s\n", resp.VaaId)
}

func runNotaryGetDelayedMessage(cmd *cobra.Command, args []string) {
	ctx := context.Background()
	conn, c, err := getAdminClient(ctx, *clientSocketPath)
	if err != nil {
		log.Fatalf("failed to get admin client: %v", err)
	}
	defer conn.Close()

	msg := nodev1.NotaryGetDelayedMessageRequest{
		VaaId: args[0],
	}
	resp, err := c.NotaryGetDelayedMessage(ctx, &msg)
	if err != nil {
		log.Fatalf("failed to run NotaryGetDelayedMessage RPC: %s", err)
	}

	fmt.Printf("Message ID: %s\nRelease Time: %s\nDetails: %s\n", resp.VaaId, resp.ReleaseTime, resp.MessageDetails)
}

func runNotaryGetBlackholedMessage(cmd *cobra.Command, args []string) {
	ctx := context.Background()
	conn, c, err := getAdminClient(ctx, *clientSocketPath)
	if err != nil {
		log.Fatalf("failed to get admin client: %v", err)
	}
	defer conn.Close()

	msg := nodev1.NotaryGetBlackholedMessageRequest{
		VaaId: args[0],
	}
	resp, err := c.NotaryGetBlackholedMessage(ctx, &msg)
	if err != nil {
		log.Fatalf("failed to run NotaryGetBlackholedMessage RPC: %s", err)
	}

	fmt.Printf("Message ID: %s\nDetails: %s\n", resp.VaaId, resp.MessageDetails)
}

func runNotaryListDelayedMessages(cmd *cobra.Command, args []string) {
	ctx := context.Background()
	conn, c, err := getAdminClient(ctx, *clientSocketPath)
	if err != nil {
		log.Fatalf("failed to get admin client: %v", err)
	}
	defer conn.Close()

	msg := nodev1.NotaryListDelayedMessagesRequest{}
	resp, err := c.NotaryListDelayedMessages(ctx, &msg)
	if err != nil {
		log.Fatalf("failed to run NotaryListDelayedMessages RPC: %s", err)
	}

	fmt.Printf("Delayed messages (%d):\n", len(resp.VaaIds))
	for _, vaaId := range resp.VaaIds {
		fmt.Println(vaaId)
	}
}

func runNotaryListBlackholedMessages(cmd *cobra.Command, args []string) {
	ctx := context.Background()
	conn, c, err := getAdminClient(ctx, *clientSocketPath)
	if err != nil {
		log.Fatalf("failed to get admin client: %v", err)
	}
	defer conn.Close()

	msg := nodev1.NotaryListBlackholedMessagesRequest{}
	resp, err := c.NotaryListBlackholedMessages(ctx, &msg)
	if err != nil {
		log.Fatalf("failed to run NotaryListBlackholedMessages RPC: %s", err)
	}

	fmt.Printf("Blackholed messages (%d):\n", len(resp.VaaIds))
	for _, vaaId := range resp.VaaIds {
		fmt.Println(vaaId)
	}
}
