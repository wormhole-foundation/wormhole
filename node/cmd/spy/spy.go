package spy

import (
	"context"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"os"
	"sync"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/p2p"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	spyv1 "github.com/certusone/wormhole/node/pkg/proto/spy/v1"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	ipfslog "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	rootCtx       context.Context
	rootCtxCancel context.CancelFunc
)

var (
	p2pNetworkID *string
	p2pPort      *uint
	p2pBootstrap *string

	statusAddr *string

	nodeKeyPath *string

	logLevel *string

	spyRPC *string
)

func init() {
	p2pNetworkID = SpyCmd.Flags().String("network", "/wormhole/dev", "P2P network identifier")
	p2pPort = SpyCmd.Flags().Uint("port", 8999, "P2P UDP listener port")
	p2pBootstrap = SpyCmd.Flags().String("bootstrap", "", "P2P bootstrap peers (comma-separated)")

	statusAddr = SpyCmd.Flags().String("statusAddr", "[::]:6060", "Listen address for status server (disabled if blank)")

	nodeKeyPath = SpyCmd.Flags().String("nodeKey", "", "Path to node key (will be generated if it doesn't exist)")

	logLevel = SpyCmd.Flags().String("logLevel", "info", "Logging level (debug, info, warn, error, dpanic, panic, fatal)")

	spyRPC = SpyCmd.Flags().String("spyRPC", "", "Listen address for gRPC interface")
}

// SpyCmd represents the node command
var SpyCmd = &cobra.Command{
	Use:   "spy",
	Short: "Run gossip spy client",
	Run:   runSpy,
}

type spyServer struct {
	spyv1.UnimplementedSpyRPCServiceServer
	logger          *zap.Logger
	subsSignedVaa   map[string]*subscriptionSignedVaa
	subsSignedVaaMu sync.Mutex
	subsAllVaa      map[string]*subscriptionAllVaa
	subsAllVaaMu    sync.Mutex
}

type message struct {
	vaaBytes []byte
}

type filterSignedVaa struct {
	chainId     vaa.ChainID
	emitterAddr vaa.Address
}
type subscriptionSignedVaa struct {
	filters []filterSignedVaa
	ch      chan message
}
type subscriptionAllVaa struct {
	filters []*spyv1.FilterEntry
	ch      chan *spyv1.SubscribeSignedVAAByTypeResponse
}

func subscriptionId() string {
	return uuid.New().String()
}

func decodeEmitterAddr(hexAddr string) (vaa.Address, error) {
	address, err := hex.DecodeString(hexAddr)
	if err != nil {
		return vaa.Address{}, status.Error(codes.InvalidArgument, fmt.Sprintf("failed to decode address: %v", err))
	}
	if len(address) != 32 {
		return vaa.Address{}, status.Error(codes.InvalidArgument, "address must be 32 bytes")
	}

	addr := vaa.Address{}
	copy(addr[:], address)

	return addr, nil
}

func (s *spyServer) PublishSignedVAA(vaaBytes []byte) error {
	s.subsSignedVaaMu.Lock()
	defer s.subsSignedVaaMu.Unlock()

	var v *vaa.VAA

	for _, sub := range s.subsSignedVaa {
		if len(sub.filters) == 0 {
			sub.ch <- message{vaaBytes: vaaBytes}
		} else {
			if v == nil {
				var err error
				v, err = vaa.Unmarshal(vaaBytes)
				if err != nil {
					return err
				}
			}

			for _, fi := range sub.filters {
				if fi.chainId == v.EmitterChain && fi.emitterAddr == v.EmitterAddress {
					sub.ch <- message{vaaBytes: vaaBytes}
				}
			}
		}
	}

	return nil
}

// TransactionIdMatches decodes both transactionIDs and checks if they are the same.
func TransactionIdMatches(batch *vaa.BatchVAA, t *spyv1.BatchFilter) (bool, error) {
	// first check if the transaction IDs match
	filterHash, err := vaa.StringToHash(t.TransactionId)
	if err != nil {
		return false, status.Error(codes.InvalidArgument, fmt.Sprintf("failed to decode filter's txId: %v", err))
	}

	matches := filterHash == batch.TransactionID
	return matches, nil
}

// BatchMatchFilter asserts that the obervation matches the values of the filter.
func BatchMatchesFilter(batch *vaa.BatchVAA, f *spyv1.BatchFilter) (bool, error) {
	// check the transaction identifier matches
	txMatch, err := TransactionIdMatches(batch, f)
	if err != nil || !txMatch {
		return false, err
	}

	// the BatchVAA's transaction ID matches the transaction ID of this filter.
	// now check if the other properties of the filter match.
	if obs := batch.Observations[0]; obs != nil {
		obsVAA := obs.Observation

		if obsVAA.EmitterChain == vaa.ChainID(f.ChainId) {
			// the emitter chain of the observation matches the filter

			if f.Nonce >= 1 {
				// filter has a nonce, so make sure it matches
				if obsVAA.Nonce != f.Nonce {
					// filter's nonce does not match the nonce of the obervation.
					return false, nil
				}
			}
			return true, nil
		}
	}

	return false, nil
}

func (s *spyServer) PublishSignedVAAByType(vaaBytes []byte) error {
	s.subsAllVaaMu.Lock()
	defer s.subsAllVaaMu.Unlock()

	// this will try to unmarshal the byte array to a VAA, and then to a BatchVAA.
	// the unmarshaling could populate one of the variable below. it's also possible
	// a new type of vaaBytes comes through, so the variables below could remain empty.

	var v *vaa.VAA
	var b *vaa.BatchVAA

	// is the byte array a VAA (v1)
	isVAA := false
	// is the byte array a BatchVAA (v2)
	isBatch := false

	v, err := vaa.Unmarshal(vaaBytes)
	// do nothing with the error, until we can try to unmarshal the bytes as a batch.
	if err != nil {
		// check if it is a batch

		// it is not a VAA, try unmarshaling to a BatchVAA
		b, err = vaa.UnmarshalBatch(vaaBytes)
		if err != nil {
			// it is not either type of VAA we know.
			// do not throw, this is not unexpected.
			// it will be returned to subscribers with no filters.
			s.logger.Warn("encountered a VAA of unknown type.",
				zap.ByteString("vaaBytes", vaaBytes))
		}
	}

	// find EmitterFilter values within the structs,
	// so they can be considered for EmitterFilter subs,
	// without being concerned about the VAA type.
	var emitterAddress vaa.Address
	var emitterChain vaa.ChainID

	// create the response(s) that will get sent out if this VAA satisfies a subscription.

	// create the top-level response struct that is agnostic to the VAA type
	var topRes *spyv1.SubscribeSignedVAAByTypeResponse

	// if v has values, the VAA unmarshal was successful.
	if v != nil && len(v.Payload) > 0 {
		isVAA = true
		emitterAddress = v.EmitterAddress
		emitterChain = v.EmitterChain

		// resData is the lowest level proto struct, it holds the byte data for whatever
		// type of response it is (VAA in this case).
		resData := &spyv1.SubscribeSignedVAAResponse{
			VaaBytes: vaaBytes,
		}

		// resType defines what struct vaa will be retuned below, res of type SignedVaa
		resType := &spyv1.SubscribeSignedVAAByTypeResponse_SignedVaa{
			SignedVaa: resData,
		}

		// topRes is the highest level proto struct, the response to the subscription
		topRes = &spyv1.SubscribeSignedVAAByTypeResponse{
			// VaaType: &spyv1.SubscribeSignedVAAByTypeResponse_SignedVaa{
			VaaType: resType,
		}

		// the proto is fully constructed ready to send.
	}

	// if b has vaules, the BatchVAA unmarshal was successful.
	if b != nil && len(b.Observations) > 0 {
		isBatch = true
		// take the EmitterAddress from the first Observation in the batch,
		// since it's not in the header of the BatchVAA.
		emitterAddress = b.Observations[0].Observation.EmitterAddress
		emitterChain = b.EmitterChain

		// resData is the lowest level proto struct, it holds the byte data for whatever
		// type of response it is (BatchVAA in this case).
		resData := &spyv1.SubscribeSignedBatchVAAResponse{
			BatchVaa: vaaBytes,
		}

		// resType defines what struct vaa will be retuned below, res of type SignedBatchVaa.
		resType := &spyv1.SubscribeSignedVAAByTypeResponse_SignedBatchVaa{
			SignedBatchVaa: resData,
		}

		// topRes is the highest level proto struct, the response to the subscription
		topRes = &spyv1.SubscribeSignedVAAByTypeResponse{
			// VaaType: &spyv1.SubscribeSignedVAAByTypeResponse_SignedVaa{
			VaaType: resType,
		}
		// proto is fully constructed ready to send.
	}

	// loop through the subscriptions and send responses to everyone that wants this VAA
	for _, sub := range s.subsAllVaa {
		if len(sub.filters) == 0 {
			// this subscription has no filters, send them the VAA.
			sub.ch <- topRes
		} else {
			// this subscription has filters.

			if !isVAA && !isBatch {
				// if the vaaBytes are of an unknown type, it won't match any filters.
				continue
			}

			for _, filterEntry := range sub.filters {
				filter := filterEntry.GetFilter()
				switch t := filter.(type) {
				case *spyv1.FilterEntry_EmitterFilter:

					filterAddr, err := decodeEmitterAddr(t.EmitterFilter.EmitterAddress)
					if err != nil {
						return status.Error(codes.InvalidArgument, fmt.Sprintf("failed to decode emitter address: %v", err))
					}
					filterChain := vaa.ChainID(t.EmitterFilter.ChainId)

					if filterChain == emitterChain && filterAddr == emitterAddress {
						// it is a match, send the response
						sub.ch <- topRes
					}

				case *spyv1.FilterEntry_BatchFilter:
					match, err := BatchMatchesFilter(b, t.BatchFilter)
					if err != nil {
						return err
					}
					if match {
						sub.ch <- topRes
					}

				case *spyv1.FilterEntry_BatchTransactionFilter:
					// make a BatchFilter struct from the BatchTransactionFilter since the latter is
					// a subset of the former's properties, so we can use TransactionIdMatches.
					batchFilter := &spyv1.BatchFilter{
						ChainId:       t.BatchTransactionFilter.ChainId,
						TransactionId: t.BatchTransactionFilter.TransactionId,
					}

					match, err := BatchMatchesFilter(b, batchFilter)
					if err != nil {
						return err
					}
					if match {
						sub.ch <- topRes
					}
				default:
					return status.Error(codes.InvalidArgument, "unsupported filter type")
				}
			}
		}
	}

	return nil
}

func (s *spyServer) SubscribeSignedVAA(req *spyv1.SubscribeSignedVAARequest, resp spyv1.SpyRPCService_SubscribeSignedVAAServer) error {
	var fi []filterSignedVaa
	if req.Filters != nil {
		for _, f := range req.Filters {
			switch t := f.Filter.(type) {
			case *spyv1.FilterEntry_EmitterFilter:
				addr, err := decodeEmitterAddr(t.EmitterFilter.EmitterAddress)
				if err != nil {
					return status.Error(codes.InvalidArgument, fmt.Sprintf("failed to decode emitter address: %v", err))
				}
				fi = append(fi, filterSignedVaa{
					chainId:     vaa.ChainID(t.EmitterFilter.ChainId),
					emitterAddr: addr,
				})
			default:
				return status.Error(codes.InvalidArgument, "unsupported filter type")
			}
		}
	}

	s.subsSignedVaaMu.Lock()
	id := subscriptionId()
	sub := &subscriptionSignedVaa{
		ch:      make(chan message, 1),
		filters: fi,
	}
	s.subsSignedVaa[id] = sub
	s.subsSignedVaaMu.Unlock()

	defer func() {
		s.subsSignedVaaMu.Lock()
		defer s.subsSignedVaaMu.Unlock()
		delete(s.subsSignedVaa, id)
	}()

	for {
		select {
		case <-resp.Context().Done():
			return resp.Context().Err()
		case msg := <-sub.ch:
			if err := resp.Send(&spyv1.SubscribeSignedVAAResponse{
				VaaBytes: msg.vaaBytes,
			}); err != nil {
				return err
			}
		}
	}
}

// SubscribeSignedVAAByType fields requests for subscriptions. Each new subscription adds a channel and request params (filters)
// to the map of active subscriptions.
func (s *spyServer) SubscribeSignedVAAByType(req *spyv1.SubscribeSignedVAAByTypeRequest, resp spyv1.SpyRPCService_SubscribeSignedVAAByTypeServer) error {
	var fi []*spyv1.FilterEntry
	if req.Filters != nil {
		for _, f := range req.Filters {
			switch t := f.Filter.(type) {

			case *spyv1.FilterEntry_EmitterFilter:
				// validate the emitter address is valid by decoding it
				_, err := decodeEmitterAddr(t.EmitterFilter.EmitterAddress)
				if err != nil {
					return status.Error(codes.InvalidArgument, fmt.Sprintf("failed to decode emitter address: %v", err))
				}
				fi = append(fi, &spyv1.FilterEntry{Filter: t})

			case *spyv1.FilterEntry_BatchFilter:
				// validate the TransactionId is valid by decoding it.
				_, err := vaa.StringToHash(t.BatchFilter.TransactionId)
				if err != nil {
					return status.Error(codes.InvalidArgument, fmt.Sprintf("failed to decode filter's txId: %v", err))
				}
				fi = append(fi, &spyv1.FilterEntry{Filter: t})

			case *spyv1.FilterEntry_BatchTransactionFilter:
				// validate the TransactionId is valid by decoding it.
				_, err := vaa.StringToHash(t.BatchTransactionFilter.TransactionId)
				if err != nil {
					return status.Error(codes.InvalidArgument, fmt.Sprintf("failed to decode filter's txId: %v", err))
				}
				fi = append(fi, &spyv1.FilterEntry{Filter: t})
			default:
				return status.Error(codes.InvalidArgument, "unsupported filter type")
			}
		}
	}

	s.subsAllVaaMu.Lock()
	id := subscriptionId()
	sub := &subscriptionAllVaa{
		ch:      make(chan *spyv1.SubscribeSignedVAAByTypeResponse, 1),
		filters: fi,
	}
	s.subsAllVaa[id] = sub
	s.subsAllVaaMu.Unlock()

	defer func() {
		s.subsAllVaaMu.Lock()
		defer s.subsAllVaaMu.Unlock()
		delete(s.subsAllVaa, id)
	}()

	for {
		select {
		case <-resp.Context().Done():
			return resp.Context().Err()
		case msg := <-sub.ch:
			if err := resp.Send(msg); err != nil {
				return err
			}
		}
	}
}

func newSpyServer(logger *zap.Logger) *spyServer {
	return &spyServer{
		logger:        logger.Named("spyserver"),
		subsSignedVaa: make(map[string]*subscriptionSignedVaa),
		subsAllVaa:    make(map[string]*subscriptionAllVaa),
	}
}

func spyServerRunnable(s *spyServer, logger *zap.Logger, listenAddr string) (supervisor.Runnable, *grpc.Server, error) {
	l, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to listen: %w", err)
	}

	logger.Info("publicrpc server listening", zap.String("addr", l.Addr().String()))

	grpcServer := common.NewInstrumentedGRPCServer(logger)
	spyv1.RegisterSpyRPCServiceServer(grpcServer, s)

	return supervisor.GRPCServer(grpcServer, l, false), grpcServer, nil
}

func runSpy(cmd *cobra.Command, args []string) {
	common.SetRestrictiveUmask()

	lvl, err := ipfslog.LevelFromString(*logLevel)
	if err != nil {
		fmt.Println("Invalid log level")
		os.Exit(1)
	}

	logger := ipfslog.Logger("wormhole-spy").Desugar()

	ipfslog.SetAllLoggers(lvl)

	// Status server
	if *statusAddr != "" {
		router := mux.NewRouter()

		router.Handle("/metrics", promhttp.Handler())

		go func() {
			logger.Info("status server listening on [::]:6060")
			logger.Error("status server crashed", zap.Error(http.ListenAndServe(*statusAddr, router)))
		}()
	}

	// Verify flags

	if *nodeKeyPath == "" {
		logger.Fatal("Please specify --nodeKey")
	}
	if *p2pBootstrap == "" {
		logger.Fatal("Please specify --bootstrap")
	}

	// Node's main lifecycle context.
	rootCtx, rootCtxCancel = context.WithCancel(context.Background())
	defer rootCtxCancel()

	// Outbound gossip message queue
	sendC := make(chan []byte)

	// Inbound observations
	obsvC := make(chan *gossipv1.SignedObservation, 50)

	// Inbound observation requests
	obsvReqC := make(chan *gossipv1.ObservationRequest, 50)

	// Inbound signed VAAs
	signedInC := make(chan *gossipv1.SignedVAAWithQuorum, 50)

	// Guardian set state managed by processor
	gst := common.NewGuardianSetState(nil)

	// RPC server
	s := newSpyServer(logger)
	rpcSvc, _, err := spyServerRunnable(s, logger, *spyRPC)
	if err != nil {
		logger.Fatal("failed to start RPC server", zap.Error(err))
	}

	// Ignore observations
	go func() {
		for {
			select {
			case <-rootCtx.Done():
				return
			case <-obsvC:
			}
		}
	}()

	// Ignore observation requests
	// Note: without this, the whole program hangs on observation requests
	go func() {
		for {
			select {
			case <-rootCtx.Done():
				return
			case <-obsvReqC:
			}
		}
	}()

	// Log signed VAAs
	go func() {
		for {
			select {
			case <-rootCtx.Done():
				return
			case v := <-signedInC:
				logger.Info("Received signed VAA",
					zap.Any("vaa", v.Vaa))
				if err := s.PublishSignedVAA(v.Vaa); err != nil {
					logger.Error("failed to publish signed VAA", zap.Error(err))
				}
				if err := s.PublishSignedVAAByType(v.Vaa); err != nil {
					logger.Error("failed to publish signed VAA by type", zap.Error(err))
				}
			}
		}
	}()

	// Load p2p private key
	var priv crypto.PrivKey
	priv, err = common.GetOrCreateNodeKey(logger, *nodeKeyPath)
	if err != nil {
		logger.Fatal("Failed to load node key", zap.Error(err))
	}

	// Run supervisor.
	supervisor.New(rootCtx, logger, func(ctx context.Context) error {
		if err := supervisor.Run(ctx, "p2p", p2p.Run(obsvC, obsvReqC, nil, sendC, signedInC, priv, nil, gst, *p2pPort, *p2pNetworkID, *p2pBootstrap, "", false, rootCtxCancel, nil, nil, nil)); err != nil {
			return err
		}

		if err := supervisor.Run(ctx, "spyrpc", rpcSvc); err != nil {
			return err
		}

		logger.Info("Started internal services")

		<-ctx.Done()
		return nil
	},
		// It's safer to crash and restart the process in case we encounter a panic,
		// rather than attempting to reschedule the runnable.
		supervisor.WithPropagatePanic)

	<-rootCtx.Done()
	logger.Info("root context cancelled, exiting...")
	// TODO: wait for things to shut down gracefully
}
